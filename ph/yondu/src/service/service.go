package service

import (
	"database/sql"
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	amqp_driver "github.com/streadway/amqp"

	"github.com/vostrok/operator/ph/yondu/src/config"
	"github.com/vostrok/utils/amqp"
	"github.com/vostrok/utils/rec"
	"time"
)

var svc Service

type EventNotify struct {
	EventName string     `json:"event_name,omitempty"`
	EventData rec.Record `json:"event_data,omitempty"`
}
type Service struct {
	api               *Yondu
	SendConsentCh     <-chan amqp_driver.Delivery
	VerifyTransCodeCh <-chan amqp_driver.Delivery
	ChargeCh          <-chan amqp_driver.Delivery
	MTCh              <-chan amqp_driver.Delivery
	consumer          Consumers
	notifier          *amqp.Notifier
	db                *sql.DB
	conf              config.ServiceConfig
}

type Consumers struct {
	SendConsent     *amqp.Consumer
	VerifyTransCode *amqp.Consumer
	Charge          *amqp.Consumer
	MT              *amqp.Consumer
}

func InitService(
	serverConfig config.ServerConfig,
	yConf config.YonduConfig,
	consumerConfig amqp.ConsumerConfig,
	notifierConfig amqp.NotifierConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = config.ServiceConfig{
		Server:   serverConfig,
		Consumer: consumerConfig,
		Notifier: notifierConfig,
		Yondu:    yConf,
	}
	svc.notifier = amqp.NewNotifier(notifierConfig)
	svc.api = initYondu(yConf)

	// process consumer
	q := svc.conf.Yondu.Queue
	svc.consumer = Consumers{
		SendConsent:     amqp.NewConsumer(consumerConfig, q.SendConsent.Name, q.SendConsent.PrefetchCount),
		VerifyTransCode: amqp.NewConsumer(consumerConfig, q.VerifyTransCode.Name, q.VerifyTransCode.PrefetchCount),
		Charge:          amqp.NewConsumer(consumerConfig, q.Charge.Name, q.Charge.PrefetchCount),
		MT:              amqp.NewConsumer(consumerConfig, q.MT.Name, q.MT.PrefetchCount),
	}
	if err := svc.consumer.SendConsent.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}
	if err := svc.consumer.VerifyTransCode.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}
	if err := svc.consumer.Charge.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}
	if err := svc.consumer.MT.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}
	amqp.InitQueue(
		svc.consumer.SendConsent,
		svc.SendConsentCh,
		processSentConsent,
		q.SendConsent.ThreadsCount,
		q.SendConsent.Name,
		q.SendConsent.Name,
	)
	amqp.InitQueue(
		svc.consumer.VerifyTransCode,
		svc.VerifyTransCodeCh,
		processVerifyTransCode,
		q.VerifyTransCode.ThreadsCount,
		q.VerifyTransCode.Name,
		q.VerifyTransCode.Name,
	)
	amqp.InitQueue(
		svc.consumer.Charge,
		svc.ChargeCh,
		processCharge,
		q.Charge.ThreadsCount,
		q.Charge.Name,
		q.Charge.Name,
	)
	amqp.InitQueue(
		svc.consumer.MT,
		svc.MTCh,
		processMT,
		q.SendConsent.ThreadsCount,
		q.SendConsent.Name,
		q.SendConsent.Name,
	)
}
func logResponse(t rec.Record, yResp YonduResponse, begin time.Time, err error) {
	fields := log.Fields{
		"yonduResponse": fmt.Sprintf("%#v", yResp),
		"tid":           t.Tid,
		"msisdn":        t.Msisdn,
		"took":          time.Since(begin),
	}
	errStr := ""
	if err != nil {
		errStr = err.Error()
		fields["error"] = errStr
	}
	svc.api.responseLog.WithFields(fields).Println("y")
}

func (svc *Service) publishResponse(data interface{}) error {
	event := amqp.EventNotify{
		EventName: "response",
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Yondu.Queue.CallBack, 0, body})
	return nil
}

func (svc *Service) publishMOResponse(eventName string, data interface{}) error {
	event := amqp.EventNotify{
		EventName: eventName,
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Yondu.Queue.MO, 0, body})
	return nil
}

func (svc *Service) publishTransactionLog(eventName string, r rec.Record) error {
	event := amqp.EventNotify{
		EventName: eventName,
		EventData: r,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Yondu.Queue.TransactionLog, 0, body})
	return nil
}
