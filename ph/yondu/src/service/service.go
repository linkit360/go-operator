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
	MT *amqp.Consumer
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

	// MT consumer
	q := svc.conf.Yondu.Queue
	svc.consumer = Consumers{
		MT: amqp.NewConsumer(consumerConfig, q.MT.Name, q.MT.PrefetchCount),
	}
	if err := svc.consumer.MT.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}

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
	yRespJSON, _ := json.Marshal(yResp)
	tJSON, _ := json.Marshal(t)
	fields := log.Fields{
		"yonduResponse": string(yRespJSON),
		"rec":           string(tJSON),
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

func logIncoming(reponseType string, params interface{}) {
	respJson, _ := json.Marshal(params)
	fields := log.Fields{
		"params": respJson,
	}
	svc.YonduAPI.responseLog.WithFields(fields).Println(reponseType)
}

func (svc *Service) publishDN(data DNParameters) error {
	event := amqp.EventNotify{
		EventName: "dn",
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Yondu.Queue.DN, 0, body, event.EventName})
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
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Yondu.Queue.MO, 0, body, event.EventName})
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
		Notice:           t.Notice,
		ServiceId:        t.ServiceId,
		SubscriptionId:   t.SubscriptionId,
		CampaignId:       t.CampaignId,
		RequestBody:      yr.RequestUrl,
		ResponseBody:     fmt.Sprintf("%v", yr.ResponseRawBody),
		ResponseDecision: yr.Yondu.Message,
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
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Yondu.Queue.TransactionLog, 0, body, event.EventName})
	return nil
}
