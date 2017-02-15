package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	m "github.com/vostrok/operator/th/qrtech/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	rec "github.com/vostrok/utils/rec"
)

func AddDNHandler(r *gin.Engine) {
	r.Group("/api").Group("/dn").POST("", svc.API.dn)
}

func (qr *QRTech) dn(c *gin.Context) {
	var err error
	r := rec.Record{
		Tid:         rec.GenerateTID(),
		CountryCode: qr.conf.CountryCode,
	}
	logCtx := log.WithField("tid", r.Tid)

	operatorCode, ok := c.GetPostForm("operator")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("cann't find operator")
		r.OperatorErr = r.OperatorErr + " no operator"
	}
	switch operatorCode {
	case "1":
		operatorCode = qr.conf.MCC + qr.conf.AisMNC
		m.AisSuccess.Inc()
	case "2":
		operatorCode = qr.conf.MCC + qr.conf.DtacMNC
		m.DtacSuccess.Inc()
	case "3":
		operatorCode = qr.conf.MCC + qr.conf.TruehMNC
		m.TruehSuccess.Inc()
	default:
		m.Errors.Inc()
		m.UnknownOperator.Inc()
	}
	r.OperatorCode, err = strconv.ParseInt(operatorCode, 10, 64)
	if err != nil {
		m.UnknownOperator.Inc()
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{
			"error": err.Error(),
			"code":  operatorCode,
		}).Error("cannot parse operator code")
	}

	shortCode, ok := c.GetPostForm("shortcode")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{}).Error("cann't find shortcode")
		r.OperatorErr = r.OperatorErr + " ;no shortcode;"
	}
	if len(shortCode) > 0 {
		serviceId, err := strconv.ParseInt(shortCode, 10, 64)
		if err != nil {
			m.Errors.Inc()
			m.WrongServiceKey.Inc()
			logCtx.WithFields(log.Fields{
				"serviceKey": shortCode,
			}).Error("wrong service key")
		} else {
			r.ServiceId = serviceId
			service, err := inmem_client.GetServiceById(serviceId)
			if err != nil {
				m.Errors.Inc()

				logCtx.WithFields(log.Fields{
					"serviceKey": shortCode,
					"service_id": serviceId,
				}).Error("cannot get service by id")
			} else {
				r.Price = int(service.Price)
				r.DelayHours = service.DelayHours
				r.PaidHours = service.PaidHours
				r.KeepDays = service.KeepDays
				r.Periodic = true
				r.PeriodicDays = r.PeriodicDays
				r.PeriodicAllowedFromHours = service.PeriodicAllowedFrom
				r.PeriodicAllowedToHours = service.PeriodicAllowedTo
			}
			campaign, err := inmem_client.GetCampaignByServiceId(serviceId)
			if err != nil {
				m.Errors.Inc()
				logCtx.WithFields(log.Fields{
					"serviceKey": shortCode,
					"service_id": serviceId,
				}).Error("cannot get campaign by service id")
			} else {
				r.CampaignId = campaign.Id
			}
		}
	} else {
		m.Errors.Inc()
		m.WrongServiceKey.Inc()
		logCtx.WithFields(log.Fields{
			"serviceKey": shortCode,
		}).Error("wrong service key")
	}
	r.OperatorToken, ok = c.GetPostForm("dnid")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("cann't find dnid")
		r.OperatorErr = r.OperatorErr + " no dnid"
	}
	var notice string
	dnerrorcode, ok := c.GetPostForm("dnerrorcode")
	if !ok {
		m.AbsentParameter.Inc()
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

	r.Msisdn, ok = c.GetPostForm("msisdn")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("cann't find msisdn")
		r.OperatorErr = r.OperatorErr + " no msn"
	}
	bcdate, ok := c.GetPostForm("bcdate")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Warn("cann't find bcdate")
		r.OperatorErr = r.OperatorErr + " no bcdate"
	}
	operatorTime, err := time.Parse("20060102", bcdate)
	if err != nil {
		m.Errors.Inc()

		err = fmt.Errorf("time.Parse: %s", err.Error())
		logCtx.WithFields(log.Fields{
			"error": err.Error(),
		}).Warn("cann't parse time")

		r.OperatorErr = r.OperatorErr + " cann't parse time"
		operatorTime = time.Now()
	}

	keyWord, ok := c.GetPostForm("keyword")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{}).Warn("cann't find keyword")
		r.OperatorErr = r.OperatorErr + " no keyword"
	}
	f := log.Fields{
		"dnid":         r.OperatorToken,
		"msisdn":       r.Msisdn,
		"shortCode":    shortCode,
		"operatorCode": operatorCode,
		"bcdate":       bcdate,
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
		OperatorTime:     operatorTime,
		OperatorCode:     r.OperatorCode,
		CountryCode:      r.CountryCode,
		Error:            r.OperatorErr,
		Price:            r.Price,
		ServiceId:        r.ServiceId,
		CampaignId:       r.CampaignId,
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
