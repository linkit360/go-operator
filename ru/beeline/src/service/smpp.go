package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	smpp_client "github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	inmem_service "github.com/vostrok/inmem/service"
	"github.com/vostrok/operator/ru/beeline/src/config"
	m "github.com/vostrok/operator/ru/beeline/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	"github.com/vostrok/utils/amqp"
	rec "github.com/vostrok/utils/rec"
	"strconv"
)

type SmppMessage struct {
	Fields    map[string]string           `json:"fields"`
	TlvFields map[pdufield.TLVType]string `json:"tlv_types"`
	Headers   pdu.Header
}

func (svc *Service) pduHandler(p pdu.Body) {
	m.Incoming.Inc()

	var err error
	f := p.Fields()
	h := p.Header()

	r := &rec.Record{
		Tid:           rec.GenerateTID(),
		CountryCode:   svc.conf.Beeline.CountryCode,
		OperatorCode:  svc.conf.Beeline.MccMnc,
		Msisdn:        string(f[pdufield.SourceAddr].Bytes()),
		OperatorToken: strconv.FormatUint(uint64(h.Seq), 10),
		Notice:        string(f[pdufield.ShortMessage].Bytes()),
	}
	defer logRequests(r.Tid, p, err)

	_, err = resolveRec(string(f[pdufield.DestinationAddr].Bytes()), r)
	if err != nil {
		return
	}
	tlv := p.TLVFields()

	switch string(tlv[pdufield.SourcePort].Bytes()) {
	case "3": // subscription enabled
	case "4": // subscritpion disabled
	case "5": // charge notify. charged <sum> rub. Next date ...
	case "6": // block_subscribtion - unsubscribe
		unsubscribeAll(*r)
	case "7": // block subscriber
	case "9": // msisdn change
	}
}

func resolveRec(dstAddress string, r *rec.Record) (
	service inmem_service.Service,
	err error,
) {
	logCtx := log.WithField("tid", r.Tid)

	if dstAddress == "" {
		m.AbsentParameter.Inc()
		m.Errors.Inc()

		err = fmt.Errorf("dstAddr required%s", "")
		logCtx.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cann't process")
		return
	}
	id := strings.Split(dstAddress, "#")
	if len(id) < 2 {
		m.WrongServiceKey.Inc()
		m.Errors.Inc()

		err = errors.New("no # in dst address")
		logCtx.WithFields(log.Fields{
			"error": err.Error(),
			"dst":   dstAddress,
		}).Error("cann't process")
		return
	}
	serviceToken := id[0]

	campaign, err := inmem_client.GetCampaignByKeyWord(serviceToken)
	if err != nil {
		m.Errors.Inc()
		err = fmt.Errorf("inmem_client.GetCampaignByKeyWord: %s", err.Error())

		logCtx.WithFields(log.Fields{
			"serviceToken": serviceToken,
			"error":        err.Error(),
		}).Error("cannot find campaign by serviceToken")

		return
	}

	r.CampaignId = campaign.Id
	r.ServiceId = campaign.ServiceId
	service, err = inmem_client.GetServiceById(campaign.ServiceId)
	if err != nil {
		m.Errors.Inc()

		err = fmt.Errorf("inmem_client.GetServiceById: %s", err.Error())
		logCtx.WithFields(log.Fields{
			"service_id": campaign.ServiceId,
			"error":      err.Error(),
		}).Error("cannot get service by id")
		return
	}
	r.Price = int(service.Price)
	r.DelayHours = service.DelayHours
	r.PaidHours = service.PaidHours
	r.KeepDays = service.KeepDays
	r.Periodic = false
	return
}

func (svc *Service) transactionLog(p pdu.Body, r *rec.Record) {
	sentAt := time.Now().UTC()
	f := p.Fields()
	tlv := p.TLVFields()

	body, _ := json.Marshal(p)

	tl := transaction_log_service.OperatorTransactionLog{
		Tid:              r.Tid,
		Msisdn:           r.Msisdn,
		OperatorToken:    r.OperatorToken,
		OperatorCode:     r.OperatorCode,
		CountryCode:      r.CountryCode,
		Error:            r.OperatorErr,
		Price:            r.Price,
		ServiceId:        r.ServiceId,
		CampaignId:       r.CampaignId,
		RequestBody:      string(body),
		ResponseBody:     "",
		ResponseDecision: string(tlv[pdufield.SourcePort].Bytes()),
		ResponseCode:     200,
		SentAt:           sentAt,
		Notice:           f[pdufield.ShortMessage].String(),
		Type:             "smpp",
	}
	svc.publishTransactionLog(tl)
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
func initTransceiver(conf config.SmppConfig, receiverFn func(p pdu.Body)) *smpp_client.Transceiver {
	smppTransceiver := &smpp_client.Transceiver{
		Addr:        conf.Addr,
		User:        conf.User,
		Passwd:      conf.Password,
		RespTimeout: time.Duration(conf.Timeout) * time.Second,
		Handler:     receiverFn,
	}
	connStatus := smppTransceiver.Bind()
	go func() {
		for c := range connStatus {
			if c.Status().String() != "Connected" {
				m.SMPPConnected.Set(0)
				log.WithFields(log.Fields{
					"user":   conf.User,
					"status": c.Status().String(),
					"error":  "disconnected: " + c.Error().Error(),
				}).Error("smpp connect failed")
			} else {
				log.WithFields(log.Fields{
					"status": c.Status().String(),
				}).Info("smpp connect ok")
				m.SMPPConnected.Set(1)
			}
		}
	}()
	log.Info("smpp transmitter init done")
	return smppTransceiver
}
