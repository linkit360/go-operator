package service

import (
	"database/sql"
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	amqp_driver "github.com/streadway/amqp"

	"github.com/vostrok/db"
	"github.com/vostrok/operator/pk/mobilink"
	"github.com/vostrok/operator/template/src/config"
	"github.com/vostrok/rabbit"
)

var svc Service

type Operator interface {
}

type Service struct {
	consumer  *rabbit.Consumer
	publisher rabbit.AMQPService
	records   <-chan amqp_driver.Delivery
	db        *sql.DB
	m         Metrics
	conf      Config
}
type Config struct {
	server    config.ServerConfig
	db        db.DataBaseConfig
	queues    config.QueueConfig
	consumer  rabbit.ConsumerConfig
	publisher rabbit.NotifierConfig
}

func InitService(
	serverConfig config.ServerConfig,
	dbConf db.DataBaseConfig,
	queuesConfig config.QueueConfig,
	consumerConfig rabbit.ConsumerConfig,
	publisherConfig rabbit.NotifierConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = Config{
		server:    serverConfig,
		db:        dbConf,
		queues:    queuesConfig,
		consumer:  consumerConfig,
		publisher: publisherConfig,
	}

	svc.db = db.Init(dbConf)

	svc.m = initMetrics()

	svc.publisher = rabbit.NewNotifier(publisherConfig, rabbit.initPublisherMetrics())

	// process consumer
	svc.consumer = rabbit.NewConsumer(consumerConfig)
	if err := svc.consumer.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}
	var err error
	svc.records, err = svc.consumer.AnnounceQueue(queuesConfig.In, queuesConfig.In)
	if err != nil {
		log.WithFields(log.Fields{
			"queue": queuesConfig.In,
			"error": err.Error(),
		}).Fatal("rbmq consumer: AnnounceQueue")
	}
	go svc.consumer.Handle(
		svc.records,
		process,
		serverConfig.ThreadsCount,
		queuesConfig.In,
		queuesConfig.In,
	)
}

func publish(eventName string, data interface{}) error {
	event := rabbit.EventNotify{
		EventName: eventName,
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.publisher.Publish(rabbit.AMQPMessage{svc.conf.queues.Out, body})
	return nil
}
