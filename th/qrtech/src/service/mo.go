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

func AddMOHandler(r *gin.Engine) {
	r.Group("/api").Group("/mo").POST("", svc.API.mo)
}

//
// POST http://www.CPurl.com/receiver.php HTTP/1.1
// HOST: CPHost
// msgid=_msgid123&msisdn=66819197088&message=P1&shortcode=4219112&motoken =
// cp123&productid=&operator=1&keyword=P1
func (qr *QRTech) mo(c *gin.Context) {
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
				r.PeriodicDays = service.PeriodicDays
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
	r.OperatorToken, ok = c.GetPostForm("msgid")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("cann't find msgid")
		r.OperatorErr = r.OperatorErr + " no msgid"
	}
	r.Msisdn, ok = c.GetPostForm("msisdn")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("cann't find msisdn")
		r.OperatorErr = r.OperatorErr + " no msn"
	}
	notice, ok := c.GetPostForm("message")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Warn("cann't find message")
		r.OperatorErr = r.OperatorErr + " no message"
	}
	keyWord, ok := c.GetPostForm("keyword")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Warn("cann't find keyword")
		r.OperatorErr = r.OperatorErr + " no keyword"
	}
	f := log.Fields{
		"operatorCode": operatorCode,
		"shortCode":    shortCode,
		"msgid":        r.OperatorToken,
		"msisdn":       r.Msisdn,
		"message":      notice,
		"keyword":      keyWord,
	}
	logCtx.WithFields(f).Info("access")

	moToken, ok := c.GetPostForm("motoken")
	if !ok {
		m.UnAuthorized.Inc()
		m.Errors.Inc()
		log.WithFields(log.Fields{"error": "unauthorized"}).Error("cann't find motoken")
		c.JSON(403, struct{}{})
		return
	}
	if moToken != qr.conf.MoToken {
		m.UnAuthorized.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{"error": "unauthorized"}).Error("motoken differs")
		c.JSON(403, struct{}{})
		return
	}

	r.SentAt = time.Now().UTC()

	logRequests("mo", f, r)
	fieldsBody, _ := json.Marshal(f)
	tl := transaction_log_service.OperatorTransactionLog{
		Tid:              r.Tid,
		Msisdn:           r.Msisdn,
		OperatorToken:    r.OperatorToken,
		OperatorCode:     r.OperatorCode,
		OperatorTime:     time.Now().UTC(),
		CountryCode:      r.CountryCode,
		Error:            r.OperatorErr,
		Price:            r.Price,
		ServiceId:        r.ServiceId,
		CampaignId:       r.CampaignId,
		RequestBody:      c.Request.URL.Path + "?" + string(fieldsBody),
		ResponseBody:     "",
		ResponseDecision: "",
		ResponseCode:     200,
		SentAt:           r.SentAt,
		Notice:           notice,
		Type:             "mo",
	}
	svc.publishTransactionLog(tl)

	if strings.Contains(keyWord, "STOP") {
		if err := svc.publishUnsubscrube(qr.conf.Queue.Unsubscribe, r); err != nil {
			m.Errors.Inc()

			logCtx.WithFields(log.Fields{
				"p":     fmt.Sprintf("%#v", r),
				"error": err.Error(),
			}).Error("sent unsubscribe failed")
		} else {
			logCtx.WithFields(log.Fields{
				"msisdn": r.Msisdn,
				"tid":    r.Tid,
				"ref":    r.OperatorToken,
			}).Info("sent unsubscribe")
		}
		return
	}

	if err := svc.publishMO(qr.conf.Queue.MO, r); err != nil {
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{
			"p":     fmt.Sprintf("%#v", r),
			"error": err.Error(),
		}).Error("sent mo failed")
	} else {
		logCtx.WithFields(log.Fields{
			"msisdn": r.Msisdn,
			"tid":    r.Tid,
			"ref":    r.OperatorToken,
		}).Info("sent mo")
	}
	m.Success.Inc()
}
