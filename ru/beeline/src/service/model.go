package service

import (
	"encoding/json"
	"fmt"

	"time"

	log "github.com/Sirupsen/logrus"
	smpp_client "github.com/fiorix/go-smpp/smpp"

	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	inmem_client "github.com/vostrok/inmem/rpcclient"
	"github.com/vostrok/operator/ru/beeline/src/config"
	m "github.com/vostrok/operator/ru/beeline/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	"github.com/vostrok/utils/amqp"
	logger "github.com/vostrok/utils/log"
	"github.com/vostrok/utils/rec"
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
	svc := Service{
		responseLog: logger.GetFileLogger(beeConf.TransactionLogFilePath.ResponseLogPath),
		requestLog:  logger.GetFileLogger(beeConf.TransactionLogFilePath.RequestLogPath),
	}
	svc.conf = config.ServiceConfig{
		Consumer: consumerConfig,
		Notifier: notifierConfig,
		Beeline:  beeConf,
	}
	svc.transceiver = initTransceiver(beeConf.SMPP, svc.pduHandler)
	svc.notifier = amqp.NewNotifier(notifierConfig)

	if err := inmem_client.Init(inMemConfig); err != nil {
		log.WithField("error", err.Error()).Fatal("cannot init inmem client")
	}
}

func logRequests(tid string, p pdu.Body, err string) {
	f := p.Fields()
	tlv := p.TLVFields()
	fields := log.Fields{
		"tid":           tid,
		"seq":           p.Header().Seq,
		"source_port":   string(tlv[pdufield.SourcePort].Bytes()),
		"source":        f[pdufield.SourceAddr].String(),
		"dst":           f[pdufield.DestinationAddr].String(),
		"short_message": f[pdufield.ShortMessage].String(),
	}
	if err != "" {
		fields["error"] = err
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
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Beeline.Queue.TransactionLog, 0, body})
	return nil
}

func OnExit() {
	defer svc.transceiver.Close()
}
