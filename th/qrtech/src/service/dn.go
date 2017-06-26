package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	mid_client "github.com/linkit360/go-mid/rpcclient"
	m "github.com/linkit360/go-operator/th/qrtech/src/metrics"
	transaction_log_service "github.com/linkit360/go-qlistener/src/service"
	rec "github.com/linkit360/go-utils/rec"
)

func AddDNHandler(r *gin.Engine) {
	r.Group("/api").Group("/dn").POST("", svc.API.dn)
}

func (qr *QRTech) dn(c *gin.Context) {
	var err error
	r := rec.Record{
		CountryCode: qr.conf.CountryCode,
		SentAt:      time.Now().UTC(),
	}

	var ok bool
	r.Msisdn, ok = c.GetPostForm("msisdn")
	if !ok {
		m.DN.AbsentParameter.Inc()
		m.Errors.Inc()
		log.WithFields(log.Fields{
			"req": c.Request.URL.String(),
		}).Error("cann't find msisdn")
		r.OperatorErr = r.OperatorErr + " no msn"
	}
	r.Tid = rec.GenerateTID(r.Msisdn)
	logCtx := log.WithField("tid", r.Tid)

	operatorCode, ok := c.GetPostForm("operator")
	if !ok {
		m.DN.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("cann't find operator")
		r.OperatorErr = r.OperatorErr + " no operator"
	}
	switch operatorCode {
	case "1":
		operatorCode = qr.conf.MCC + qr.conf.AisMNC
		m.DN.AisSuccess.Inc()
	case "2":
		operatorCode = qr.conf.MCC + qr.conf.DtacMNC
		m.DN.DtacSuccess.Inc()
	case "3":
		operatorCode = qr.conf.MCC + qr.conf.TruehMNC
		m.DN.TruehSuccess.Inc()
	default:
		m.Errors.Inc()
		m.DN.UnknownOperator.Inc()
	}
	r.OperatorCode, err = strconv.ParseInt(operatorCode, 10, 64)
	if err != nil {
		m.DN.UnknownOperator.Inc()
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{
			"error": err.Error(),
			"code":  operatorCode,
		}).Error("cannot parse operator code")
	}

	r.ServiceCode, ok = c.GetPostForm("shortcode")
	if !ok {
		m.DN.AbsentParameter.Inc()
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{}).Error("cann't find shortcode")
		r.OperatorErr = r.OperatorErr + " ;no shortcode;"
	}
	if len(r.ServiceCode) > 0 {
		service, err := mid_client.GetServiceByCode(r.ServiceCode)
		if err != nil {
			m.Errors.Inc()

			logCtx.WithFields(log.Fields{
				"serviceKey": r.ServiceCode,
			}).Error("cannot get service by code")
		} else {
			r.Price = service.PriceCents
			r.DelayHours = service.DelayHours
			r.PaidHours = service.PaidHours
			r.RetryDays = service.RetryDays
			r.Periodic = true
			r.PeriodicDays = r.PeriodicDays
			r.PeriodicAllowedFromHours = service.PeriodicAllowedFrom
			r.PeriodicAllowedToHours = service.PeriodicAllowedTo
		}
		campaign, err := mid_client.GetCampaignByServiceCode(r.ServiceCode)
		if err != nil {
			m.Errors.Inc()
			logCtx.WithFields(log.Fields{
				"serviceKey": r.ServiceCode,
			}).Error("cannot get campaign by service code")
		} else {
			r.CampaignId = campaign.Id
		}

	} else {
		m.Errors.Inc()
		m.DN.WrongServiceKey.Inc()
		logCtx.WithFields(log.Fields{
			"serviceKey": r.ServiceCode,
		}).Error("wrong service key")
	}
	r.OperatorToken, ok = c.GetPostForm("dnid")
	if !ok {
		m.DN.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("cann't find dnid")
		r.OperatorErr = r.OperatorErr + " no dnid"
	}
	var notice string
	dnerrorcode, ok := c.GetPostForm("dnerrorcode")
	if !ok {
		m.DN.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("cann't find dnerrorcode")
		r.OperatorErr = r.OperatorErr + " no dnerrorcode"
	} else {
		codeMessage, ok := qr.conf.DN.Code[dnerrorcode]
		if ok {
			switch {
			case strings.Contains(dnerrorcode, "200"):
				m.DN.MTSuccessfull200.Inc()
				r.Paid = true
			case strings.Contains(dnerrorcode, "100"):
				m.DN.MTSentToQueueSuccessfully100.Inc()
			case strings.Contains(dnerrorcode, "500"):
				m.DN.MTRejected500.Inc()
			case strings.Contains(dnerrorcode, "501"):
				m.DN.MessageFormatError501.Inc()
			case strings.Contains(dnerrorcode, "510"):
				m.DN.UnknownSubscriber510.Inc()
			case strings.Contains(dnerrorcode, "511"):
				m.DN.SubscriberBarred511.Inc()
			case strings.Contains(dnerrorcode, "512"):
				m.DN.SubscriberError512.Inc()
			case strings.Contains(dnerrorcode, "520"):
				m.DN.OperatorFailure520.Inc()
			case strings.Contains(dnerrorcode, "521"):
				m.DN.OperatorCongestion521.Inc()
			case strings.Contains(dnerrorcode, "530"):
				m.DN.ChargingError530.Inc()
			case strings.Contains(dnerrorcode, "531"):
				m.DN.SubscriberNotEnoughBalance531.Inc()
			case strings.Contains(dnerrorcode, "532"):
				m.DN.SubscriberExceededFrequency532.Inc()
			case strings.Contains(dnerrorcode, "550"):
				m.DN.OtherError550.Inc()
			default:
				m.DN.UnknownCode.Inc()
			}
			notice = dnerrorcode + ": " + codeMessage
		} else {
			m.DN.UnknownCode.Inc()
			notice = dnerrorcode + ": unknown dn error code"
		}
	}

	keyWord, ok := c.GetPostForm("keyword")
	if !ok {
		m.DN.AbsentParameter.Inc()
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{}).Warn("cann't find keyword")
		r.OperatorErr = r.OperatorErr + " no keyword"
	}
	f := log.Fields{
		"dnid":         r.OperatorToken,
		"msisdn":       r.Msisdn,
		"shortCode":    r.ServiceCode,
		"operatorCode": operatorCode,
		"dnerrorcode":  dnerrorcode,
		"keyword":      keyWord,
	}
	logCtx.WithFields(f).Info("access")

	fieldsBody, _ := json.Marshal(f)
	logRequests("dn", f, r)
	tl := transaction_log_service.OperatorTransactionLog{
		Tid:              r.Tid,
		Msisdn:           r.Msisdn,
		OperatorToken:    r.OperatorToken,
		OperatorTime:     time.Now().UTC(),
		OperatorCode:     r.OperatorCode,
		CountryCode:      r.CountryCode,
		Error:            r.OperatorErr,
		Price:            r.Price,
		ServiceCode:      r.ServiceCode,
		CampaignCode:     r.CampaignId,
		RequestBody:      c.Request.URL.Path + "/" + string(fieldsBody),
		ResponseBody:     "",
		ResponseDecision: "",
		ResponseCode:     200,
		SentAt:           r.SentAt,
		Notice:           notice,
		Type:             "dn",
	}
	if r.Paid {
		tl.ResponseDecision = "paid"
	}
	svc.publishTransactionLog(tl)

	if err := svc.publishDN(qr.conf.Queue.DN, r); err != nil {
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{
			"p":     fmt.Sprintf("%#v", r),
			"error": err.Error(),
		}).Error("sent mo failed")
	} else {
		logCtx.WithFields(log.Fields{
			"msisdn": r.Msisdn,
			"tid":    r.Tid,
		}).Info("sent dn")
	}
	m.Success.Inc()
}
