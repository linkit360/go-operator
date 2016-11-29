package service

import (
	"database/sql"
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	amqp_driver "github.com/streadway/amqp"

	mobilink_api "github.com/vostrok/operator/pk/mobilink/src/api"
	"github.com/vostrok/operator/pk/mobilink/src/config"
	m "github.com/vostrok/operator/pk/mobilink/src/metrics"
	"github.com/vostrok/utils/amqp"
	queue_config "github.com/vostrok/utils/config"
)

var svc Service

type Service struct {
	TarifficateChan <-chan amqp_driver.Delivery
	SMSChan         <-chan amqp_driver.Delivery
	api             *mobilink_api.Mobilink
	consumer        Consumer
	notifier        *amqp.Notifier
	db              *sql.DB
	conf            Config
}
type Config struct {
	server   config.ServerConfig
	queues   queue_config.OperatorQueueConfig
	consumer amqp.ConsumerConfig
	notifier amqp.NotifierConfig
}

type Consumer struct {
	Requests    *amqp.Consumer
	SMSRequests *amqp.Consumer
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

	m.Init()
	svc.notifier = amqp.NewNotifier(notifierConfig)
	svc.api = mobilink_api.Init(mbConf.Rps, mbConf, svc.notifier)

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

	// queue for sms requests
	amqp.InitQueue(
		svc.consumer.SMSRequests,
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
