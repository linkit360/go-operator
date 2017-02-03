package service

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	"github.com/vostrok/operator/ru/beeline/src/config"
	m "github.com/vostrok/operator/ru/beeline/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	logger "github.com/vostrok/utils/log"
	rec "github.com/vostrok/utils/rec"
)

type Beeline struct {
	conf        config.BeelineConfig
	Throttle    ThrottleConfig
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
}

type ThrottleConfig struct {
}

func AddHandlers(r *gin.Engine) {
	_ = r.Group("/api").Group("/mo")
}

// nothing to add
func AddTestHandlers(r *gin.Engine) {
	//:= r.Group("/beeline/")
}
func initBeeline(bConf config.BeelineConfig) *Beeline {
	y := &Beeline{
		conf:        bConf,
		Throttle:    ThrottleConfig{},
		responseLog: logger.GetFileLogger(bConf.TransactionLogFilePath.ResponseLogPath),
		requestLog:  logger.GetFileLogger(bConf.TransactionLogFilePath.RequestLogPath),
	}
	return y
}

type Params struct {
}

func (beeline *Beeline) mo(operator string, c *gin.Context) {
	r := rec.Record{
		Tid:          rec.GenerateTID(),
		CountryCode:  beeline.conf.CountryCode,
		OperatorCode: beeline.conf.MccMnc,
	}
	logCtx := log.WithField("tid", r.Tid)
	logCtx.WithField("url", c.Request.URL.Path+"/"+c.Request.URL.RawQuery).Info("access")

	// know campaig by service key
	serviceKey, ok := c.GetQuery("svk")
	if !ok {
		m.AbsentParameter.Inc()
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{}).Error("cann't find service key")
		r.OperatorErr = r.OperatorErr + " no service key"
	}
	if len(serviceKey) >= 4 {
		res, err := inmem_client.GetCampaignByKeyWord(serviceKey[:4])
		if err != nil {
			m.Errors.Inc()

			logCtx.WithFields(log.Fields{
				"serviceKey": serviceKey[:4],
			}).Error("cannot find campaign by service key")
		} else {
			r.CampaignId = res.Id
			r.ServiceId = res.ServiceId
			service, err := inmem_client.GetServiceById(res.ServiceId)
			if err != nil {
				m.Errors.Inc()

				logCtx.WithFields(log.Fields{
					"serviceKey": serviceKey,
					"service_id": res.ServiceId,
				}).Error("cannot get service by id")
			} else {
				r.Price = int(service.Price)
				r.DelayHours = service.DelayHours
				r.PaidHours = service.PaidHours
				r.KeepDays = service.KeepDays
				r.Periodic = false
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
		Notice:           "",
		Type:             "mo",
	}
	svc.publishTransactionLog(tl)

	if err := svc.publishMO(beeline.conf.Queue.MO, r); err != nil {
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
