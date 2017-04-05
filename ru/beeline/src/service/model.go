package service

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	smpp_client "github.com/fiorix/go-smpp/smpp"

	inmem_client "github.com/linkit360/go-inmem/rpcclient"
	"github.com/linkit360/go-operator/ru/beeline/src/config"
	m "github.com/linkit360/go-operator/ru/beeline/src/metrics"
	transaction_log_service "github.com/linkit360/go-qlistener/src/service"
	"github.com/linkit360/go-utils/amqp"
	logger "github.com/linkit360/go-utils/log"
	"github.com/linkit360/go-utils/rec"
)

var svc Service

type EventNotify struct {
	EventName string     `json:"event_name,omitempty"`
	EventData rec.Record `json:"event_data,omitempty"`
}
type Service struct {
	conf        config.ServiceConfig
	location    *time.Location
	transceiver *smpp_client.Transceiver
	responseLog *log.Logger
	requestLog  *log.Logger
	notifier    *amqp.Notifier
}

func InitService(
	beeConf config.BeelineConfig,
	inMemConfig inmem_client.ClientConfig,
	consumerConfig amqp.ConsumerConfig,
	notifierConfig amqp.NotifierConfig,
) {
	log.SetLevel(log.DebugLevel)

	if err := inmem_client.Init(inMemConfig); err != nil {
		log.WithField("error", err.Error()).Fatal("cannot init inmem client")
	}

	svc = Service{
		responseLog: logger.GetFileLogger(beeConf.TransactionLogFilePath.ResponseLogPath),
		requestLog:  logger.GetFileLogger(beeConf.TransactionLogFilePath.RequestLogPath),
		notifier:    amqp.NewNotifier(notifierConfig),
	}
	svc.conf = config.ServiceConfig{
		Consumer: consumerConfig,
		Notifier: notifierConfig,
		Beeline:  beeConf,
	}

	initTransceiver(beeConf.SMPP, pduHandler)
}

func logRequests(r rec.Record, fields log.Fields, err error) {
	recBody, _ := json.Marshal(r)
	fields["rec"] = recBody
	if err != nil {
		fields["error"] = err.Error()
	}
	svc.requestLog.WithFields(fields).Println(".")
}

func (svc *Service) newSub(r rec.Record) (err error) {
	defer func() {
		fields := log.Fields{
			"tid": r.Tid,
			"q":   svc.conf.Beeline.Queue.MO,
		}
		if err != nil {
			m.NotifyErrors.Inc()
			m.Errors.Inc()

			fields["errors"] = err.Error()
			fields["tl"] = fmt.Sprintf("%#v", r)
			log.WithFields(fields).Error("cannot enqueue")
		} else {
			log.WithFields(fields).Debug("sent")
		}
	}()
	if r.SentAt.IsZero() {
		r.SentAt = time.Now().UTC()
	}
	event := amqp.EventNotify{
		EventName: "mo",
		EventData: r,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Beeline.Queue.MO, 0, body, event.EventName})
	return nil
}

func (svc *Service) publishTransactionLog(tl transaction_log_service.OperatorTransactionLog) (err error) {
	defer func() {
		fields := log.Fields{
			"tid": tl.Tid,
			"q":   svc.conf.Beeline.Queue.TransactionLog,
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
		EventName: "smpp",
		EventData: tl,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Beeline.Queue.TransactionLog, 0, body, event.EventName})
	return nil
}

func OnExit() {
	log.WithField("pid", os.Getpid()).Info("close transiever conn")
	if svc.transceiver != nil {
		svc.transceiver.Close()
	}
}

func newSubscription(r rec.Record) error {
	r.SentAt = time.Now().UTC()
	event := EventNotify{
		EventName: "mo",
		EventData: r,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	log.Debugf("new subscription %s", body)
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Beeline.Queue.MO, 0, body, event.EventName})
	return nil
}

func chargeNotify(msg rec.Record) error {
	return _notifyDBAction("WriteTransaction", msg)
}

func unsubscribe(msg rec.Record) error {
	return _notifyDBAction("Unsubscribe", msg)
}

func unsubscribeAll(msg rec.Record) error {
	return _notifyDBAction("UnsubscribeAll", msg)
}
func _notifyDBAction(eventName string, msg rec.Record) error {
	var err error
	msg.SentAt = time.Now().UTC()

	defer func() {
		if err != nil {
			fields := log.Fields{
				"data":  fmt.Sprintf("%#v", msg),
				"q":     svc.conf.Beeline.Queue.DBActions,
				"event": eventName,
				"error": fmt.Errorf(eventName+": %s", err.Error()),
			}
			log.WithFields(fields).Error("cannot send")
		} else {
			fields := log.Fields{
				"event":    eventName,
				"tid":      msg.Tid,
				"status":   msg.SubscriptionStatus,
				"periodic": msg.Periodic,
				"q":        svc.conf.Beeline.Queue.DBActions,
			}
			log.WithFields(fields).Info("sent")
		}
	}()

	if eventName == "" {
		err = fmt.Errorf("QueueSend: %s", "Empty event name")
		return err
	}

	event := amqp.EventNotify{
		EventName: eventName,
		EventData: msg,
	}
	var body []byte
	body, err = json.Marshal(event)
	if err != nil {
		err = fmt.Errorf(eventName+" json.Marshal: %s", err.Error())
		return err
	}
	svc.notifier.Publish(amqp.AMQPMessage{
		QueueName: svc.conf.Beeline.Queue.DBActions,
		Body:      body,
		EventName: eventName,
	})
	return nil
}
