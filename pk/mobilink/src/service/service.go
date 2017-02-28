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
	mbConf mobilink_api.Config,
	qConfig config.QueueConfig,
	consumerConfig amqp.ConsumerConfig,
	notifierConfig amqp.NotifierConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = Config{
		server:   serverConfig,
		queues:   qConfig,
		consumer: consumerConfig,
		notifier: notifierConfig,
	}
	svc.notifier = amqp.NewNotifier(notifierConfig)
	svc.api = mobilink_api.Init(mbConf.Rps, mbConf, svc.notifier)

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

func (svc *Service) publishSMSResponse(eventName string, data interface{}) error {
	event := amqp.EventNotify{
		EventName: eventName,
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{queue_config.SMSResponsesQueue("mobilink"), 0, body, event.EventName})
	return nil
}
