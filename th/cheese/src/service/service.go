package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	mid_client "github.com/linkit360/go-mid/rpcclient"
	"github.com/linkit360/go-operator/th/cheese/src/config"
	m "github.com/linkit360/go-operator/th/cheese/src/metrics"
	transaction_log_service "github.com/linkit360/go-qlistener/src/service"
	"github.com/linkit360/go-utils/amqp"
	"github.com/linkit360/go-utils/rec"
)

var svc Service

type EventNotify struct {
	EventName string     `json:"event_name,omitempty"`
	EventData rec.Record `json:"event_data,omitempty"`
}
type Service struct {
	CheeseAPI *Cheese
	notifier  *amqp.Notifier
	db        *sql.DB
	conf      config.ServiceConfig
}

func InitService(
	serverConfig config.ServerConfig,
	cheeseConf config.CheeseConfig,
	consumerConfig amqp.ConsumerConfig,
	notifierConfig amqp.NotifierConfig,
	midConfig mid_client.ClientConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = config.ServiceConfig{
		Server:   serverConfig,
		Consumer: consumerConfig,
		Notifier: notifierConfig,
		Cheese:   cheeseConf,
	}
	svc.notifier = amqp.NewNotifier(notifierConfig)
	svc.CheeseAPI = initCheese(cheeseConf)

	if err := mid_client.Init(midConfig); err != nil {
		log.WithField("error", err.Error()).Fatal("cannot init mid client")
	}
}

func logRequests(requestType string, t rec.Record, req *http.Request, err string) {

	recJson, _ := json.Marshal(t)
	fields := log.Fields{
		"url":    req.URL.Path + "/" + req.URL.RawQuery,
		"rec":    string(recJson),
		"msisdn": t.Msisdn,
	}
	if err != "" {
		fields["error"] = err
	}
	svc.CheeseAPI.requestLog.WithFields(fields).Println(requestType)
}

func (svc *Service) publishMO(queue string, data interface{}) error {
	event := amqp.EventNotify{
		EventName: queue,
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{queue, 0, body, event.EventName})
	return nil
}

type Response struct {
	RequestUrl string
	Error      string
	Time       time.Time
	Decision   string
	Code       int
	RawBody    string
}

func (svc *Service) publishTransactionLog(tl transaction_log_service.OperatorTransactionLog) (err error) {
	defer func() {
		fields := log.Fields{
			"tid": tl.Tid,
			"q":   svc.conf.Cheese.Queue.TransactionLog,
		}
		if err != nil {
			m.NotifyErrors.Inc()
			m.Errors.Inc()

			fields["errors"] = err.Error()
			fields["tl"] = fmt.Sprintf("%#v", tl)
			log.WithFields(fields).Error("cannot enqueue")
		} else {
			log.WithFields(fields).Debug("sent")
		}
	}()
	if tl.SentAt.IsZero() {
		tl.SentAt = time.Now().UTC()
	}
	event := amqp.EventNotify{
		EventName: "mo",
		EventData: tl,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Cheese.Queue.TransactionLog, 0, body, event.EventName})
	return nil
}

func (svc *Service) publishUnsubscrube(r rec.Record) error {
	event := amqp.EventNotify{
		EventName: "Unsubscribe",
		EventData: r,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Cheese.Queue.Unsubscribe, 0, body, event.EventName})
	return nil
}
