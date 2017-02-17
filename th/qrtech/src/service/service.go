package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	amqp_driver "github.com/streadway/amqp"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	"github.com/vostrok/operator/th/qrtech/src/config"
	m "github.com/vostrok/operator/th/qrtech/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	"github.com/vostrok/utils/amqp"
	logger "github.com/vostrok/utils/log"
	rec "github.com/vostrok/utils/rec"
)

var svc Service

type QRTech struct {
	conf        config.QRTechConfig
	Throttle    ThrottleConfig
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
	mtConsumer  *amqp.Consumer
	mtChan      <-chan amqp_driver.Delivery
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

	qr.mtConsumer = amqp.NewConsumer(consumerConfig, qrTechConf.Queue.MT.Name, qrTechConf.Queue.MT.PrefetchCount)
	if err := qr.mtConsumer.Connect(); err != nil {
		log.Fatal("rbmq consumer connect:", err.Error())
	}
	amqp.InitQueue(
		qr.mtConsumer,
		qr.mtChan,
		processCharge,
		qrTechConf.Queue.MT.ThreadsCount,
		qrTechConf.Queue.MT.Name,
		qrTechConf.Queue.MT.Name,
	)
	return qr
}

type EventNotify struct {
	EventName string     `json:"event_name,omitempty"`
	EventData rec.Record `json:"event_data,omitempty"`
}
type Service struct {
	API      *QRTech
	notifier *amqp.Notifier
	db       *sql.DB
	conf     config.ServiceConfig
}

func InitService(
	serverConfig config.ServerConfig,
	qrtechConf config.QRTechConfig,
	consumerConfig amqp.ConsumerConfig,
	notifierConfig amqp.NotifierConfig,
	inMemConfig inmem_client.RPCClientConfig,
) {
	log.SetLevel(log.DebugLevel)
	svc.conf = config.ServiceConfig{
		Server:   serverConfig,
		Consumer: consumerConfig,
		Notifier: notifierConfig,
		QRTech:   qrtechConf,
	}
	svc.notifier = amqp.NewNotifier(notifierConfig)
	svc.API = initQRTech(qrtechConf, consumerConfig)

	if err := inmem_client.Init(inMemConfig); err != nil {
		log.WithField("error", err.Error()).Fatal("cannot init inmem client")
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
	svc.notifier.Publish(amqp.AMQPMessage{queue, 0, body})
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
	svc.notifier.Publish(amqp.AMQPMessage{queue, 0, body})
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
	svc.notifier.Publish(amqp.AMQPMessage{queue, 0, body})
	return nil
}

func logRequests(requestType string, fields log.Fields, t rec.Record) {
	recJson, _ := json.Marshal(t)
	fields["type"] = requestType
	fields["rec"] = string(recJson)
	svc.API.requestLog.WithFields(fields).Println(requestType)
}

func logResponse(
	requestType string,
	t rec.Record,
	tl transaction_log_service.OperatorTransactionLog,
	err error,
) {

	recJson, _ := json.Marshal(t)
	fields := log.Fields{
		"type":     requestType,
		"req":      tl.RequestBody,
		"body":     tl.ResponseBody,
		"status":   tl.ResponseCode,
		"desicion": tl.ResponseDecision,
		"rec":      string(recJson),
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	svc.API.requestLog.WithFields(fields).Println(requestType)
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
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.QRTech.Queue.TransactionLog, 0, body})
	return nil
}
