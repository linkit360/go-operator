package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	mid_client "github.com/linkit360/go-mid/rpcclient"
	"github.com/linkit360/go-operator/th/qrtech/src/config"
	m "github.com/linkit360/go-operator/th/qrtech/src/metrics"
	transaction_log_service "github.com/linkit360/go-qlistener/src/service"
	"github.com/linkit360/go-utils/amqp"
	logger "github.com/linkit360/go-utils/log"
	rec "github.com/linkit360/go-utils/rec"
)

var svc Service

type QRTech struct {
	conf        config.QRTechConfig
	Throttle    ThrottleConfig
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
}

type ThrottleConfig struct {
	MT <-chan time.Time
}

func initQRTech(qrTechConf config.QRTechConfig, consumerConfig amqp.ConsumerConfig) *QRTech {
	qr := &QRTech{
		conf:        qrTechConf,
		client:      &http.Client{Timeout: time.Duration(qrTechConf.MT.Timeout) * time.Second},
		Throttle:    ThrottleConfig{MT: time.Tick(time.Second / time.Duration(qrTechConf.MT.RPS+1))},
		responseLog: logger.GetFileLogger(qrTechConf.TransactionLogFilePath.ResponseLogPath),
		requestLog:  logger.GetFileLogger(qrTechConf.TransactionLogFilePath.RequestLogPath),
	}
	var err error
	qr.location, err = time.LoadLocation(qrTechConf.Location)
	if err != nil {
		log.WithFields(log.Fields{
			"location": qrTechConf.Location,
			"error":    err,
		}).Fatal("location")
	}
	go qr.sendMT()
	return qr
}

type EventNotify struct {
	EventName string     `json:"event_name,omitempty"`
	EventData rec.Record `json:"event_data,omitempty"`
}
type Service struct {
	API       *QRTech
	notifier  *amqp.Notifier
	db        *sql.DB
	conf      config.ServiceConfig
	internals *config.InternalsConfig
}

func InitService(
	serverConfig config.ServerConfig,
	qrtechConf config.QRTechConfig,
	consumerConfig amqp.ConsumerConfig,
	notifierConfig amqp.NotifierConfig,
	midConfig mid_client.ClientConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = config.ServiceConfig{
		Server:   serverConfig,
		Consumer: consumerConfig,
		Notifier: notifierConfig,
		QRTech:   qrtechConf,
	}
	svc.internals = &config.InternalsConfig{}
	svc.notifier = amqp.NewNotifier(notifierConfig)
	svc.API = initQRTech(qrtechConf, consumerConfig)

	if err := mid_client.Init(midConfig); err != nil {
		log.WithField("error", err.Error()).Fatal("cannot init mid client")
	}
}

func (svc *Service) publishMO(queue string, r rec.Record) error {
	event := amqp.EventNotify{
		EventName: queue,
		EventData: r,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{queue, 0, body, event.EventName})
	return nil
}
func (svc *Service) publishDN(queue string, r rec.Record) error {
	event := amqp.EventNotify{
		EventName: queue,
		EventData: r,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{queue, 0, body, event.EventName})
	return nil
}
func (svc *Service) publishUnsubscrube(queue string, data interface{}) error {
	event := amqp.EventNotify{
		EventName: "Unsubscribe",
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	svc.notifier.Publish(amqp.AMQPMessage{queue, 0, body, event.EventName})
	return nil
}

func logRequests(requestType string, fields log.Fields, t rec.Record) {
	recJson, _ := json.Marshal(t)
	fields["type"] = requestType
	fields["rec"] = string(recJson)
	svc.API.requestLog.WithFields(fields).Println(requestType)
	log.WithFields(fields).Info(requestType)
}

func (svc *Service) publishTransactionLog(tl transaction_log_service.OperatorTransactionLog) (err error) {
	defer func() {
		fields := log.Fields{
			"tid": tl.Tid,
			"q":   svc.conf.QRTech.Queue.TransactionLog,
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
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.QRTech.Queue.TransactionLog, 0, body, event.EventName})
	return nil
}
