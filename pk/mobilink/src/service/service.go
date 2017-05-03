package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	amqp_driver "github.com/streadway/amqp"

	"github.com/gin-gonic/gin"
	"github.com/linkit360/go-operator/pk/mobilink/src/config"
	transaction_log_service "github.com/linkit360/go-qlistener/src/service"
	"github.com/linkit360/go-utils/amqp"
	queue_config "github.com/linkit360/go-utils/config"
	logger "github.com/linkit360/go-utils/log"
)

var svc Service

type Service struct {
	TarifficateChan <-chan amqp_driver.Delivery
	SMSChan         <-chan amqp_driver.Delivery
	api             *mobilink
	consumer        Consumer
	notifier        *amqp.Notifier
	db              *sql.DB
	conf            ServiceConfig
}
type ServiceConfig struct {
	server   config.ServerConfig
	queues   config.QueueConfig
	consumer amqp.ConsumerConfig
	notifier amqp.NotifierConfig
}

type Consumer struct {
	Requests    *amqp.Consumer
	SMSRequests *amqp.Consumer
}

func InitService(
	serverConfig config.ServerConfig,
	mbConf config.MobilinkConfig,
	qConfig config.QueueConfig,
	consumerConfig amqp.ConsumerConfig,
	notifierConfig amqp.NotifierConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = ServiceConfig{
		server:   serverConfig,
		queues:   qConfig,
		consumer: consumerConfig,
		notifier: notifierConfig,
	}
	svc.notifier = amqp.NewNotifier(notifierConfig)
	svc.api = initMobilink(mbConf.Rps, mbConf, svc.notifier)

	// process consumer
	svc.consumer = Consumer{
		Requests:    amqp.NewConsumer(consumerConfig, qConfig.Requests.Name, qConfig.Requests.PrefetchCount),
		SMSRequests: amqp.NewConsumer(consumerConfig, qConfig.SMSRequests.Name, qConfig.SMSRequests.PrefetchCount),
	}
	if err := svc.consumer.Requests.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}
	if err := svc.consumer.SMSRequests.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}

	// queue for tarifficate requests
	amqp.InitQueue(
		svc.consumer.Requests,
		svc.TarifficateChan,
		processTarifficate,
		qConfig.Requests.ThreadsCount,
		qConfig.Requests.Name,
		qConfig.Requests.Name,
	)

	// queue for sms requests
	amqp.InitQueue(
		svc.consumer.SMSRequests,
		svc.SMSChan,
		processSMS,
		qConfig.SMSRequests.ThreadsCount,
		qConfig.SMSRequests.Name,
		qConfig.SMSRequests.Name,
	)

}

type mobilink struct {
	conf        config.MobilinkConfig
	rps         int
	ThrottleMT  <-chan time.Time
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
	notifier    *amqp.Notifier
}

// prefix from table
func My(msisdn string) bool {
	return msisdn[:2] == "92"
}

func initMobilink(
	mobilinkRps int,
	mobilinkConf config.MobilinkConfig,
	notifier *amqp.Notifier,
) *mobilink {

	mb := &mobilink{
		rps:  mobilinkRps,
		conf: mobilinkConf,
	}

	mb.notifier = notifier

	mb.ThrottleMT = time.Tick(time.Second / time.Duration(mobilinkConf.Rps+1))
	mb.requestLog = logger.GetFileLogger(mobilinkConf.TransactionLog.RequestLogPath)

	mb.responseLog = logger.GetFileLogger(mobilinkConf.TransactionLog.ResponseLogPath)

	var err error
	mb.location, err = time.LoadLocation(mobilinkConf.Location)
	if err != nil {
		log.WithFields(log.Fields{
			"location": mobilinkConf.Location,
			"error":    err,
		}).Fatal("init location")
	}

	mb.client = &http.Client{
		Timeout: time.Duration(mobilinkConf.MT.TimeoutSec) * time.Second,
	}
	log.Info("mobilink init done")

	return mb
}

func (svc *Service) publishResponse(eventName string, data interface{}) error {
	event := amqp.EventNotify{
		EventName: eventName,
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{queue_config.ResponsesQueue("mobilink"), 0, body, event.EventName})
	return nil
}

func (svc *Service) notifyTransactionLog(eventName string, tl transaction_log_service.OperatorTransactionLog) error {
	tl.SentAt = time.Now().UTC()
	event := amqp.EventNotify{
		EventName: eventName,
		EventData: tl,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.queues.OperatorTransactionLog, 0, body, event.EventName})
	return nil
}

func AddMobilinkTestHandlers(r *gin.Engine) {
	rgMobilink := r.Group("/mobilink", AccessHandler)
	rgMobilink.POST("/paid", paidHandler)
	rgMobilink.GET("/sms", paidHandler)
	rgMobilink.POST("/failed", failedHandler)
	rgMobilink.POST("/postpaid", postPaidHandler)

}
func paidHandler(c *gin.Context) {
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte(`<value><i4>0</i4></value>`))
}
func failedHandler(c *gin.Context) {
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte(`<value><i4>112</i4></value>`))
}
func postPaidHandler(c *gin.Context) {
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte(`<value><i4>11</i4></value>`))
}

func AccessHandler(c *gin.Context) {
	c.Next()
	log.WithFields(log.Fields{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"req":    c.Request.URL.RawQuery,
	}).Info("access")
}
