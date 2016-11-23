package service

import (
	"database/sql"
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	amqp_driver "github.com/streadway/amqp"

	mobilink_api "github.com/vostrok/operator/pk/mobilink/src/api"
	"github.com/vostrok/operator/pk/mobilink/src/config"
	"github.com/vostrok/utils/amqp"
	queue_config "github.com/vostrok/utils/config"
	m "github.com/vostrok/utils/metrics"
)

var svc Service

type Service struct {
	TarifficateChan <-chan amqp_driver.Delivery
	SMSChan         <-chan amqp_driver.Delivery
	api             *mobilink_api.Mobilink
	consumer        *amqp.Consumer
	notifier        *amqp.Notifier
	db              *sql.DB
	m               Metrics
	conf            Config
}
type Config struct {
	server   config.ServerConfig
	queues   queue_config.OperatorQueueConfig
	consumer amqp.ConsumerConfig
	notifier amqp.NotifierConfig
}

type Metrics struct {
	Dropped m.Gauge
	Empty   m.Gauge
}

func initMetrics() Metrics {
	m := Metrics{
		Dropped: m.NewGauge("", "", "dropped", "mobilink queue dropped"),
		Empty:   m.NewGauge("", "", "empty", "mobilink queue empty"),
	}
	return m
}

func InitService(
	serverConfig config.ServerConfig,
	mbConf mobilink_api.Config,
	queuesConfig queue_config.OperatorQueueConfig,
	consumerConfig amqp.ConsumerConfig,
	notifierConfig amqp.NotifierConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = Config{
		server:   serverConfig,
		queues:   queuesConfig,
		consumer: consumerConfig,
		notifier: notifierConfig,
	}

	svc.m = initMetrics()
	svc.notifier = amqp.NewNotifier(notifierConfig)
	svc.api = mobilink_api.Init(mbConf.Rps, mbConf, svc.notifier)

	// process consumer
	svc.consumer = amqp.NewConsumer(consumerConfig)
	if err := svc.consumer.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}

	// queue for tarifficate requests
	amqp.InitQueue(
		svc.consumer,
		svc.TarifficateChan,
		processTarifficate,
		serverConfig.ThreadsCount,
		queuesConfig.Requests,
		queuesConfig.Requests,
	)

	// queue for sms requests
	amqp.InitQueue(
		svc.consumer,
		svc.SMSChan,
		processSMS,
		serverConfig.ThreadsCount,
		queuesConfig.SMSRequest,
		queuesConfig.SMSRequest,
	)

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
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.queues.Responses, body})
	return nil
}

func (svc *Service) publishSMSResponse(eventName string, data interface{}) error {
	event := amqp.EventNotify{
		EventName: eventName,
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.queues.SMSResponse, body})
	return nil
}
