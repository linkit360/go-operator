package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	"github.com/vostrok/operator/th/cheese/src/config"
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
	inMemConfig inmem_client.RPCClientConfig,
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

	if err := inmem_client.Init(inMemConfig); err != nil {
		log.WithField("error", err.Error()).Fatal("cannot init inmem client")
	}
}

func logRequests(requestType string, t rec.Record, req *http.Request, begin *time.Time, err error) {

	fields := log.Fields{
		"url":    req.URL.Path + "/" + req.URL.RawQuery,
		"rec":    fmt.Sprintf("%#v", t),
		"msisdn": t.Msisdn,
		"took":   time.Since(begin),
	}
	errStr := ""
	if err != nil {
		errStr = err.Error()
		fields["error"] = errStr
	} else {
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
	svc.notifier.Publish(amqp.AMQPMessage{queue, 0, body})
	return nil
}

type Response struct {
	RequestUrl string
	Error      string
	Time       *time.Time
	Decision   string
	Code       int
	RawBody    string
}

func (svc *Service) publishTransactionLog(eventName string, t rec.Record, resp Response) error {
	tl := transaction_log_service.OperatorTransactionLog{
		Tid:              t.Tid,
		Msisdn:           t.Msisdn,
		OperatorToken:    t.OperatorToken,
		OperatorCode:     t.OperatorCode,
		CountryCode:      t.CountryCode,
		Error:            resp.Error,
		Price:            t.Price,
		ServiceId:        t.ServiceId,
		SubscriptionId:   t.SubscriptionId,
		CampaignId:       t.CampaignId,
		RequestBody:      resp.RequestUrl,
		ResponseBody:     resp.RawBody,
		ResponseDecision: resp.Decision,
		ResponseCode:     resp.Code,
		SentAt:           resp.Time,
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
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Cheese.Queue.TransactionLog, 0, body})
	return nil
}
