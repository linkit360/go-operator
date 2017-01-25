package service

import (
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	"github.com/vostrok/operator/th/cheese/src/config"
	logger "github.com/vostrok/utils/log"
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
	_ := r.Group("/cheese/")
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
	Ref        string `json:"ref"`
	Msisdn     string `json:"msisdn"`
	ServiceKey string `json:"svk"`
	Acs        string `json:"acs"`
	Chn        string `json:"chn"`
	Mdt        string `json:"time"`
	Sub        string `json:"subordinary"`
}

func (cheese *Cheese) Ais(c *gin.Context) {
	cheese.mo("ais", c)
}

func (cheese *Cheese) Dtac(c *gin.Context) {
	cheese.mo("dtac", c)
}

func (cheese *Cheese) Trueh(c *gin.Context) {
	cheese.mo("trueh", c)
}

// ref=61201000015999095326&msn=669XXXXXXX0&svk=4504XXXXX&acs= REG_SUCCESS&chn=SMS&mdt=2016-12-01 00:00:18.227
func (cheese *Cheese) mo(subordinary string, c *gin.Context) {

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
