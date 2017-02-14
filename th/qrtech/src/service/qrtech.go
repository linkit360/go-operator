package service

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	"github.com/vostrok/operator/th/qrtech/src/config"
	m "github.com/vostrok/operator/th/qrtech/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	logger "github.com/vostrok/utils/log"
	rec "github.com/vostrok/utils/rec"
	"strings"
)

type QRTech struct {
	conf        config.QRTechConfig
	Throttle    ThrottleConfig
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
}

type ThrottleConfig struct {
}

func AddHandlers(r *gin.Engine) {
	r.Group("/api").Group("/mo").POST("", svc.API.mo)
}

// MT handler
func AddTestHandlers(r *gin.Engine) {
	//:= r.Group("/qr/mt")
}

func initQRTech(yConf config.QRTechConfig) *QRTech {
	y := &QRTech{
		conf:        yConf,
		Throttle:    ThrottleConfig{},
		responseLog: logger.GetFileLogger(yConf.TransactionLogFilePath.ResponseLogPath),
		requestLog:  logger.GetFileLogger(yConf.TransactionLogFilePath.RequestLogPath),
	}
	return y
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

	shortCode, ok := c.GetQuery("shortcode")
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
				r.Periodic = false
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
	notice, ok := c.GetQuery("message")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Warn("cann't find message")
		r.OperatorErr = r.OperatorErr + " no message"
	}
	keyWord, ok := c.GetQuery("keyword")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Warn("cann't find keyword")
		r.OperatorErr = r.OperatorErr + " no keyword"
	}
	logCtx.WithFields(log.Fields{
		"operatorCode": operatorCode,
		"shortCode":    shortCode,
		"msgid":        r.OperatorToken,
		"msisdn":       r.Msisdn,
		"message":      notice,
		"keyword":      keyWord,
	}).Info("access")

	moToken, ok := c.GetQuery("motoken")
	if !ok {
		m.UnAuthorized.Inc()
		m.Errors.Inc()
		log.WithFields(log.Fields{}).Error("unauthorized")
		c.JSON(403, struct{}{})
		return
	}
	if moToken != qr.conf.MoToken {
		m.UnAuthorized.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("unauthorized")
		c.JSON(403, struct{}{})
		return
	}

	r.SentAt = time.Now().UTC()

	logRequests("mo", r, c.Request, r.OperatorErr)
	tl := transaction_log_service.OperatorTransactionLog{
		Tid:              r.Tid,
		Msisdn:           r.Msisdn,
		OperatorToken:    r.OperatorToken,
		OperatorCode:     r.OperatorCode,
		CountryCode:      r.CountryCode,
		Error:            r.OperatorErr,
		Price:            r.Price,
		ServiceId:        r.ServiceId,
		CampaignId:       r.CampaignId,
		RequestBody:      c.Request.URL.Path + "/" + c.Request.URL.RawQuery,
		ResponseBody:     "",
		ResponseDecision: "",
		ResponseCode:     200,
		SentAt:           r.SentAt,
		Notice:           notice,
		Type:             "mo",
	}
	svc.publishTransactionLog(tl)

	r.Pixel, ok = c.GetQuery("aff_sub")
	if ok && len(r.Pixel) >= 4 {
		log.WithFields(log.Fields{}).Debug("found pixel")
		svc.notifyPixel(r)
	}

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

// just log and count all requests
func AccessHandler(c *gin.Context) {
	begin := time.Now()
	c.Next()
	responseTime := time.Since(begin)

	path := c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		path = path + "?" + c.Request.URL.RawQuery

	}
	fields := log.Fields{
		"method": c.Request.Method,
		"url":    path,
		"since":  responseTime,
	}
	if len(c.Errors) > 0 {
		fields["error"] = c.Errors.String()
		log.WithFields(fields).Error("failed")
	} else {
		log.WithFields(fields).Info("access")
	}
}
