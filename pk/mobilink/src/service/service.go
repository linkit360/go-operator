package service

import (
	"database/sql"

	log "github.com/Sirupsen/logrus"
	amqp_driver "github.com/streadway/amqp"

	"github.com/vostrok/db"
	"github.com/vostrok/operator/pk/mobilink/src/config"
	"github.com/vostrok/rabbit"
)

var svc Service

type Service struct {
	consumer *rabbit.Consumer
	records  <-chan amqp_driver.Delivery
	db       *sql.DB
	m        Metrics
	conf     Config
}
type Config struct {
	server   config.ServerConfig
	db       db.DataBaseConfig
	consumer rabbit.ConsumerConfig
}

func InitService(
	serverConfig config.ServerConfig,
	dbConf db.DataBaseConfig,
	consumerConfig rabbit.ConsumerConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = Config{
		server:   serverConfig,
		db:       dbConf,
		consumer: consumerConfig,
	}

	svc.db = db.Init(dbConf)
	svc.m = initMetrics()

	// process consumer
	svc.consumer = rabbit.NewConsumer(consumerConfig)
	if err := svc.consumer.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}

	var err error
	svc.records, err = svc.consumer.AnnounceQueue(serverConfig.Queue, serverConfig.Queue)
	if err != nil {
		log.WithFields(log.Fields{
			"queue": serverConfig.Queue,
			"error": err.Error(),
		}).Fatal("rbmq consumer: AnnounceQueue")
	}
	go svc.consumer.Handle(svc.records, process, serverConfig.ThreadsCount, serverConfig.Queue, serverConfig.Queue)
}
