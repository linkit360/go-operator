package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	amqp_driver "github.com/streadway/amqp"

	yondu_api "github.com/vostrok/operator/ph/yondu/src/api"
	yondu_config "github.com/vostrok/operator/pk/yondu/src/config"
	"github.com/vostrok/utils/amqp"
	queue_config "github.com/vostrok/utils/config"
	m "github.com/vostrok/utils/metrics"
)

var svc Service

type Service struct {
	api             *yondo_api.API
	TarifficateChan <-chan amqp_driver.Delivery
	SMSChan         <-chan amqp_driver.Delivery
	consumer        Consumer
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
		Dropped: m.NewGauge("", "", "dropped", "yondo queue dropped"),
		Empty:   m.NewGauge("", "", "empty", "yondo queue empty"),
	}

	go func() {
		for range time.Tick(time.Minute) {
			m.Dropped.Update()
			m.Empty.Update()
		}
	}()
	return m
}

type Consumer struct {
	Requests    *amqp.Consumer
	SMSRequests *amqp.Consumer
}

func InitService(
	serverConfig config.ServerConfig,
	mbConf yondo_api.Config,
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
	svc.api = yondo_api.Init(mbConf.Rps, mbConf, svc.notifier)

	// process consumer
	svc.consumer = Consumer{
		Requests:    amqp.NewConsumer(consumerConfig, queuesConfig.Requests),
		SMSRequests: amqp.NewConsumer(consumerConfig, queuesConfig.SMSRequest),
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
		serverConfig.ThreadsCount,
		queuesConfig.Requests,
		queuesConfig.Requests,
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
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.queues.Responses, 0, body})
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
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.queues.SMSResponse, 0, body})
	return nil
}
