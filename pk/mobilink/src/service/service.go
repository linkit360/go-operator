package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	amqp_driver "github.com/streadway/amqp"

	"github.com/vostrok/db"
	m "github.com/vostrok/metrics"
	mobilink_api "github.com/vostrok/operator/pk/mobilink/src/api"
	"github.com/vostrok/operator/pk/mobilink/src/config"
	"github.com/vostrok/rabbit"
)

var svc Service

type Service struct {
	api      *mobilink_api.Mobilink
	consumer *rabbit.Consumer
	notifier *rabbit.Notifier
	records  <-chan amqp_driver.Delivery
	db       *sql.DB
	m        Metrics
	conf     Config
}
type Config struct {
	server   config.ServerConfig
	db       db.DataBaseConfig
	queues   config.QueueConfig
	consumer rabbit.ConsumerConfig
	notifier rabbit.NotifierConfig
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
	go func() {
		for range time.Tick(time.Minute) {
			m.Dropped.Update()
			m.Empty.Update()
		}
	}()
	return m
}

func InitService(
	serverConfig config.ServerConfig,
	mbConf mobilink_api.Config,
	dbConf db.DataBaseConfig,
	queuesConfig config.QueueConfig,
	consumerConfig rabbit.ConsumerConfig,
	notifierConfig rabbit.NotifierConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = Config{
		server:   serverConfig,
		db:       dbConf,
		queues:   queuesConfig,
		consumer: consumerConfig,
		notifier: notifierConfig,
	}

	svc.db = db.Init(dbConf)

	svc.m = initMetrics()

	svc.api = mobilink_api.Init(mbConf.Rps, mbConf, svc.notifier)

	svc.notifier = rabbit.NewNotifier(notifierConfig)

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

func (svc *Service) publishResponse(eventName string, data interface{}) error {
	event := rabbit.EventNotify{
		EventName: eventName,
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(rabbit.AMQPMessage{svc.conf.queues.Out, body})
	return nil
}
