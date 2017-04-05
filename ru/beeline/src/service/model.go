package service

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	smpp_client "github.com/fiorix/go-smpp/smpp"

	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
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
	svc = Service{
		responseLog: logger.GetFileLogger(beeConf.TransactionLogFilePath.ResponseLogPath),
		requestLog:  logger.GetFileLogger(beeConf.TransactionLogFilePath.RequestLogPath),
	}
	svc.conf = config.ServiceConfig{
		Consumer: consumerConfig,
		Notifier: notifierConfig,
		Beeline:  beeConf,
	}
	initTransceiver(beeConf.SMPP, pduHandler)
	svc.notifier = amqp.NewNotifier(notifierConfig)

	if err := inmem_client.Init(inMemConfig); err != nil {
		log.WithField("error", err.Error()).Fatal("cannot init inmem client")
	}
}

func logRequests(tid string, p pdu.Body, err error) {
	f := p.Fields()
	h := p.Header()
	tlv := p.TLVFields()
	fields := log.Fields{
		"id":          h.ID,
		"len":         h.Len,
		"seq":         h.Seq,
		"status":      h.Status,
		"msisdn":      field(f, pdufield.SourceAddr),
		"sn":          field(f, pdufield.ShortMessage),
		"dst":         field(f, pdufield.DestinationAddr),
		"source_port": tlvfield(tlv, pdufield.SourcePort),
	}

	if err != nil {
		fields["error"] = err.Error()
	}
	svc.requestLog.WithFields(fields).Println(".")
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
		EventName: "mo",
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

func unsubscribeAll(msg rec.Record) error {
	return _notifyDBAction("UnsubscribeAll", &msg)
}
func _notifyDBAction(eventName string, msg *rec.Record) (err error) {
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
		return
	}

	event := amqp.EventNotify{
		EventName: eventName,
		EventData: msg,
	}
	var body []byte
	body, err = json.Marshal(event)

	if err != nil {
		err = fmt.Errorf(eventName+" json.Marshal: %s", err.Error())
		return
	}
	svc.notifier.Publish(amqp.AMQPMessage{
		QueueName: svc.conf.Beeline.Queue.DBActions,
		Body:      body,
	})
	return nil
}
