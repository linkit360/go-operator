package service

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	"github.com/vostrok/operator/th/cheese/src/config"
	m "github.com/vostrok/operator/th/cheese/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	logger "github.com/vostrok/utils/log"
	rec "github.com/vostrok/utils/rec"
)

type Cheese struct {
	conf        config.CheeseConfig
	Throttle    ThrottleConfig
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
}

type ThrottleConfig struct {
}

func AddHandlers(r *gin.Engine) {
	tgMOAPI := r.Group("/api").Group("/mo")
	tgMOAPI.Group("/ais").GET("", AccessHandler, svc.CheeseAPI.Ais)
	tgMOAPI.Group("/dtac").GET("", AccessHandler, svc.CheeseAPI.Dtac)
	tgMOAPI.Group("/trueh").GET("", AccessHandler, svc.CheeseAPI.Trueh)
}

// nothing to add
func AddTestHandlers(r *gin.Engine) {
	//:= r.Group("/cheese/")
}
func initCheese(yConf config.CheeseConfig) *Cheese {
	y := &Cheese{
		conf:        yConf,
		Throttle:    ThrottleConfig{},
		responseLog: logger.GetFileLogger(yConf.TransactionLogFilePath.ResponseLogPath),
		requestLog:  logger.GetFileLogger(yConf.TransactionLogFilePath.RequestLogPath),
	}
	return y
}

type Params struct {
	Ref        string `json:"operator_token"`
	Msisdn     string `json:"msisdn"`
	ServiceKey string `json:"service_key"`
	Acs        string `json:"acs"`
	Channel    string `json:"channel"`
	DateTime   string `json:"time"`
	Operator   string `json:"operator"`
}

// /api/mo/ais?ref=70127201524999064995&msn=66870443662&svk=450435203&acs=UNREG_IMMEDIATE&chn=IVR&mdt=2017-01-27%2020:15:28.463
func (cheese *Cheese) Ais(c *gin.Context) {
	cheese.mo("ais", c)
}

// api/mo/dtac?ref=200327131724321&svk=450435201&msn=66619921971&chn=5.CC(CRM)&acs=unregister&mdt=2017-01-27%2020:17:28.396
func (cheese *Cheese) Dtac(c *gin.Context) {
	cheese.mo("dtac", c)
}

func (cheese *Cheese) Trueh(c *gin.Context) {
	cheese.mo("trueh", c)
}

func (cheese *Cheese) mo(operator string, c *gin.Context) {
	operatorCode := ""
	switch operator {
	case "ais":
		operatorCode = cheese.conf.MCC + cheese.conf.AisMNC
		m.AisSuccess.Inc()
	case "dtac":
		operatorCode = cheese.conf.MCC + cheese.conf.DtacMNC
		m.DtacSuccess.Inc()
	case "trueh":
		operatorCode = cheese.conf.MCC + cheese.conf.TruehMNC
		m.TruehSuccess.Inc()
	default:
		m.Errors.Inc()
		m.UnknownOperator.Inc()
	}
	r := rec.Record{
		Tid:         rec.GenerateTID(),
		CountryCode: cheese.conf.CountryCode,
	}
	logCtx := log.WithField("tid", r.Tid)
	logCtx.WithField("url", c.Request.URL.Path+"/"+c.Request.URL.RawQuery).Info("access")

	var err error
	r.OperatorCode, err = strconv.ParseInt(operatorCode, 10, 64)
	if err != nil {
		m.UnknownOperator.Inc()
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{
			"error": err.Error(),
			"code":  operatorCode,
		}).Error("cannot parse operator code")
	}

	serviceKey, ok := c.GetQuery("svk")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{}).Error("cann't find service key")
		r.OperatorErr = r.OperatorErr + " no service key"
	}
	if len(serviceKey) > 0 {
		serviceId, err := strconv.ParseInt(serviceKey, 10, 64)
		if err != nil {
			m.Errors.Inc()
			m.WrongServiceKey.Inc()
			logCtx.WithFields(log.Fields{
				"serviceKey": serviceKey,
			}).Error("wrong service key")
		} else {
			r.ServiceId = serviceId
			service, err := inmem_client.GetServiceById(serviceId)
			if err != nil {
				m.Errors.Inc()

				logCtx.WithFields(log.Fields{
					"serviceKey": serviceKey,
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
					"serviceKey": serviceKey,
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
			"serviceKey": serviceKey,
		}).Error("wrong service key")
	}
	r.OperatorToken, ok = c.GetQuery("ref")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("cann't find operator token")
		r.OperatorErr = r.OperatorErr + " no ref"
	}
	r.Msisdn, ok = c.GetQuery("msn")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Error("cann't find msisdn")
		r.OperatorErr = r.OperatorErr + " no msn"
	}
	notice, ok := c.GetQuery("chn")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Warn("cann't find channel")
		r.OperatorErr = r.OperatorErr + " no chn"
	}
	_, ok = c.GetQuery("acs")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Warn("cann't find acs")
		r.OperatorErr = r.OperatorErr + " no acs"
	}
	dateTime, ok := c.GetQuery("mdt")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()
		logCtx.WithFields(log.Fields{}).Warn("cann't find date and time")
		r.OperatorErr = r.OperatorErr + " no mdt"
	}

	r.SentAt, err = time.Parse("2006-01-02 15:04:05.999", dateTime)
	if err != nil {
		m.MOParseTimeError.Inc()
		m.Errors.Inc()

		err = fmt.Errorf("time.Parse: %s", err.Error())
		logCtx.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot parse time")
		r.OperatorErr = r.OperatorErr + " cannot parse time"
		r.SentAt = time.Now().UTC()
		err = nil
	}
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

	if err := svc.publishMO(cheese.conf.Queue.MO, r); err != nil {
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
	c.Header("X-Response-Time", responseTime.String())
}
