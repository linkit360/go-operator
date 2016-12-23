package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	amqp_driver "github.com/streadway/amqp"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	"github.com/vostrok/operator/ph/yondu/src/config"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	"github.com/vostrok/utils/amqp"
	queue_config "github.com/vostrok/utils/config"
	"github.com/vostrok/utils/rec"
)

var svc Service

type EventNotify struct {
	EventName string     `json:"event_name,omitempty"`
	EventData rec.Record `json:"event_data,omitempty"`
}
type Service struct {
	YonduAPI          *Yondu
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
	SendConsent *amqp.Consumer
	Charge      *amqp.Consumer
	MT          *amqp.Consumer
}

func InitService(
	serverConfig config.ServerConfig,
	yConf config.YonduConfig,
	consumerConfig amqp.ConsumerConfig,
	notifierConfig amqp.NotifierConfig,
	inMemConfig inmem_client.RPCClientConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = config.ServiceConfig{
		Server:   serverConfig,
		Consumer: consumerConfig,
		Notifier: notifierConfig,
		Yondu:    yConf,
	}
	svc.notifier = amqp.NewNotifier(notifierConfig)
	svc.YonduAPI = initYondu(yConf)

	if err := inmem_client.Init(inMemConfig); err != nil {
		log.WithField("error", err.Error()).Fatal("cannot init inmem client")
	}

	// process consumer
	q := svc.conf.Yondu.Queue
	svc.consumer = Consumers{
		SendConsent: amqp.NewConsumer(consumerConfig, q.SendConsent.Name, q.SendConsent.PrefetchCount),
		Charge:      amqp.NewConsumer(consumerConfig, q.Charge.Name, q.Charge.PrefetchCount),
		MT:          amqp.NewConsumer(consumerConfig, q.MT.Name, q.MT.PrefetchCount),
	}
	if err := svc.consumer.SendConsent.Connect(); err != nil {
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
		q.MT.ThreadsCount,
		q.MT.Name,
		q.MT.Name,
	)
}

func logRequests(requestType string, t rec.Record, yResp YonduResponseExtended, begin time.Time, err error) {
	fields := log.Fields{
		"yonduResponse": fmt.Sprintf("%#v", yResp),
		"rec":           fmt.Sprintf("%#v", t),
		"msisdn":        t.Msisdn,
		"took":          time.Since(begin),
	}
	errStr := ""
	if err != nil {
		errStr = err.Error()
		fields["error"] = errStr
	} else {
	}
	svc.YonduAPI.requestLog.WithFields(fields).Println(requestType)
}
func logResponses(reponseType string, params interface{}) {
	fields := log.Fields{
		"params": fmt.Sprintf("%#v", params),
	}
	svc.YonduAPI.responseLog.WithFields(fields).Println(reponseType)
}

func (svc *Service) publishCallback(data CallbackParameters) error {
	event := amqp.EventNotify{
		EventName: "callback",
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Yondu.Queue.CallBack.Name, 0, body})
	return nil
}

func (svc *Service) publishMO(data MOParameters) error {
	event := amqp.EventNotify{
		EventName: "mo",
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Yondu.Queue.MO.Name, 0, body})
	return nil
}

func (svc *Service) publishTransactionLog(eventName string, yr YonduResponseExtended, t rec.Record) error {
	tl := transaction_log_service.OperatorTransactionLog{
		Tid:              t.Tid,
		Msisdn:           t.Msisdn,
		OperatorToken:    t.OperatorToken,
		OperatorCode:     t.OperatorCode,
		CountryCode:      t.CountryCode,
		Error:            yr.ResponseError,
		Price:            t.Price,
		ServiceId:        t.ServiceId,
		SubscriptionId:   t.SubscriptionId,
		CampaignId:       t.CampaignId,
		RequestBody:      yr.RequestUrl,
		ResponseBody:     fmt.Sprintf("%v", yr.ResponseRawBody),
		ResponseDecision: yr.Yondu.Response.Message,
		ResponseCode:     yr.ResponseCode,
		SentAt:           yr.ResponseTime,
		Type:             eventName,
	}
	tl.SentAt = time.Now().UTC()
	event := amqp.EventNotify{
		EventName: eventName,
		EventData: tl,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Yondu.Queue.TransactionLog, 0, body})
	return nil
}

func (svc *Service) newSubscriptionNotify(msg rec.Record) error {
	msg.SentAt = time.Now().UTC()
	event := EventNotify{
		EventName: "new_subscription",
		EventData: msg,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	queue := queue_config.NewSubscriptionQueueName(svc.conf.Yondu.Name)
	svc.notifier.Publish(amqp.AMQPMessage{queue, uint8(1), body})
	return nil
}
