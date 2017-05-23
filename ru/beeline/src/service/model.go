package service

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	smpp_client "github.com/fiorix/go-smpp/smpp"

	inmem_client "github.com/linkit360/go-mid/rpcclient"
	"github.com/linkit360/go-operator/ru/beeline/src/config"
	"github.com/linkit360/go-utils/amqp"
	logger "github.com/linkit360/go-utils/log"
)

var svc Service

type EventNotify struct {
	EventName string   `json:"event_name,omitempty"`
	EventData Incoming `json:"event_data,omitempty"`
}
type Service struct {
	conf        config.ServiceConfig
	location    *time.Location
	transceiver *smpp_client.Transceiver
	responseLog *log.Logger
	requestLog  *log.Logger
	notifier    *amqp.Notifier
}

func OnExit() {
	log.WithField("pid", os.Getpid()).Info("close transiever conn")
	if svc.transceiver != nil {
		svc.transceiver.Close()
	}
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

	reConnectTranceiver(beeConf.SMPP, pduHandler)
}

func publish(i Incoming) error {
	i.SentAt = time.Now().UTC()
	event := EventNotify{
		EventName: "in",
		EventData: i,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	log.Debugf("incoming %s", body)
	svc.notifier.Publish(amqp.AMQPMessage{svc.conf.Beeline.Queue.SMPPIn, 0, body, event.EventName})
	return nil
}
