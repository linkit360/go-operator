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
	m "github.com/vostrok/operator/th/cheese/src/metrics"
	pixels "github.com/vostrok/pixels/src/notifier"
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
	queues config.QueuesConfig,
	cheeseConf config.CheeseConfig,
	consumerConfig amqp.ConsumerConfig,
	notifierConfig amqp.NotifierConfig,
	inMemConfig inmem_client.RPCClientConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = config.ServiceConfig{
		Server:   serverConfig,
		Queues:   queues,
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

func logRequests(requestType string, t rec.Record, req *http.Request, err error) {

	fields := log.Fields{
		"url":    req.URL.Path + "/" + req.URL.RawQuery,
		"rec":    fmt.Sprintf("%#v", t),
		"msisdn": t.Msisdn,
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
	Time       time.Time
	Decision   string
	Code       int
	RawBody    string
}

func (svc *Service) publishTransactionLog(tl transaction_log_service.OperatorTransactionLog) (err error) {
	defer func() {
		fields := log.Fields{
			"tid": tl.Tid,
			"q":   svc.conf.Queues.TransactionLog,
		}
		if err != nil {
			m.NotifyErrors.Inc()

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
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Cheese.Queue.TransactionLog, 0, body})
	return nil
}

func (svc *Service) notifyPixel(r rec.Record) (err error) {
	msg := pixels.Pixel{
		Tid:            r.Tid,
		Msisdn:         r.Msisdn,
		CampaignId:     r.CampaignId,
		SubscriptionId: r.SubscriptionId,
		OperatorCode:   r.OperatorCode,
		CountryCode:    r.CountryCode,
		Pixel:          r.Pixel,
		Publisher:      r.Publisher,
	}

	defer func() {
		fields := log.Fields{
			"tid": msg.Tid,
			"q":   svc.conf.Queues.Pixels,
		}
		if err != nil {
			m.NotifyErrors.Inc()

			fields["errors"] = err.Error()
			fields["pixel"] = fmt.Sprintf("%#v", msg)
			log.WithFields(fields).Error("cannot enqueue")
		} else {
			log.WithFields(fields).Debug("sent")
		}
	}()
	eventName := "pixels"
	event := amqp.EventNotify{
		EventName: eventName,
		EventData: msg,
	}
	body, err := json.Marshal(event)
	if err != nil {
		err = fmt.Errorf("json.Marshal: %s", err.Error())
		return err
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Queues.Pixels, uint8(0), body})
	return nil
}
