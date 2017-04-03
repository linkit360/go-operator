package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	smpp_client "github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"

	inmem_client "github.com/linkit360/go-inmem/rpcclient"
	inmem_service "github.com/linkit360/go-inmem/service"
	"github.com/linkit360/go-operator/ru/beeline/src/config"
	m "github.com/linkit360/go-operator/ru/beeline/src/metrics"
	transaction_log_service "github.com/linkit360/go-qlistener/src/service"
	"github.com/linkit360/go-utils/amqp"
	rec "github.com/linkit360/go-utils/rec"
)

type SmppMessage struct {
	Fields    map[string]string           `json:"fields"`
	TlvFields map[pdufield.TLVType]string `json:"tlv_types"`
	Headers   pdu.Header
}

func (svc *Service) pduHandler(p pdu.Body) {
	m.Incoming.Inc()
	var b bytes.Buffer
	err := p.SerializeTo(&b)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Debug("serialize")
		return
	}
	// "\x00\x00\x00@\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x0179661904936\x00\x03\x018580#3\x00\x00\x00\x01\x00\x00\x00\x00\b\x00\f\x04!\x04B\x04>\x04?\x001\x00 "
	//"&pdu.codec{h:(*pdu.Header)(0xc420201d00), l:pdufield.List{\"service_type\", \"source_addr_ton\", \"source_addr_npi\", \"source_addr\", \"dest_add
	//r_ton\", \"dest_addr_npi\", \"destination_addr\", \"esm_class\", \"protocol_id\", \"priority_flag\", \"schedule_delivery_time\", \"validity_period\", \"registered_delivery\", \"replace_if_present_flag\",
	//\"data_coding\", \"sm_default_msg_id\", \"sm_length\", \"short_message\"}, f:pdufield.Map{\"schedule_delivery_time\":(*pdufield.Variable)(0xc42020f5a0), \"service_type\":(*pdufield.Variable)(0xc42020f540)
	//, \"source_addr_ton\":(*pdufield.Fixed)(0xc420201d10), \"source_addr_npi\":(*pdufield.Fixed)(0xc420201d11), \"source_addr\":(*pdufield.Variable)(0xc42020f560), \"dest_addr_npi\":(*pdufield.Fixed)(0xc42020
	//1d13), \"destination_addr\":(*pdufield.Variable)(0xc42020f580), \"priority_flag\":(*pdufield.Fixed)(0xc420201d32), \"sm_default_msg_id\":(*pdufield.Fixed)(0xc420201d4b), \"dest_addr_ton\":(*pdufield.Fixed
	//)(0xc420201d12), \"registered_delivery\":(*pdufield.Fixed)(0xc420201d48), \"sm_length\":(*pdufield.Fixed)(0xc420201d4c), \"short_message\":(*pdufield.SM)(0xc42020f5e0), \"protocol_id\":(*pdufield.Fixed)(0
	//xc420201d31), \"validity_period\":(*pdufield.Variable)(0xc42020f5c0), \"data_coding\":(*pdufield.Fixed)(0xc420201d4a), \"esm_class\":(*pdufield.Fixed)(0xc420201d30), \"replace_if_present_flag\":(*pdufield
	//.Fixed)(0xc420201d49)}, t:pdufield.TLVMap{0x20a:(*pdufield.TLVBody)(0xc42020f600)}}

	log.WithFields(log.Fields{
		"msgee": b.String(),
	}).Debug("process")

	//var err error
	f := p.Fields()
	h := p.Header()
	tlv := p.TLVFields()
	log.WithFields(log.Fields{
		"id":          h.ID,
		"len":         h.Len,
		"seq":         h.Seq,
		"status":      h.Status,
		"msisdn":      string(f[pdufield.SourceAddr].Bytes()),
		"sn":          string(f[pdufield.ShortMessage].Bytes()),
		"dst":         string(f[pdufield.DestinationAddr].Bytes()),
		"source_port": string(tlv[pdufield.SourcePort].Bytes()),
	}).Debug("pdu")

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

	log.WithFields(log.Fields{
		"msgee": fmt.Sprintf("%#v", p),
	}).Debug("process")

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
			"error": "dstAddress is empty",
		}).Error("cann't process")
		return
	}
	serviceToken := dstAddress[0:4]
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
	logCtx.WithFields(log.Fields{
		"campaign": fmt.Sprintf("%#v", campaign),
	}).Debug("process")

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
	logCtx.WithFields(log.Fields{
		"service": fmt.Sprintf("%#v", service),
	}).Debug("process")

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
