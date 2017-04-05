package service

import (
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
	rec "github.com/linkit360/go-utils/rec"
)

func pduHandler(p pdu.Body) {
	m.Incoming.Inc()
	var err error

	//var err error
	f := p.Fields()
	h := p.Header()
	tlv := p.TLVFields()
	log.WithFields(log.Fields{
		"id":          h.ID,
		"len":         h.Len,
		"seq":         h.Seq,
		"status":      h.Status,
		"msisdn":      field(f, pdufield.SourceAddr),
		"sn":          field(f, pdufield.ShortMessage),
		"dst":         field(f, pdufield.DestinationAddr),
		"source_port": tlvfield(tlv, pdufield.SourcePort),
	}).Debug("pdu")

	r := &rec.Record{
		Tid:           rec.GenerateTID(),
		CountryCode:   svc.conf.Beeline.CountryCode,
		OperatorCode:  svc.conf.Beeline.MccMnc,
		Msisdn:        field(f, pdufield.SourceAddr),
		OperatorToken: strconv.FormatUint(uint64(h.Seq), 10),
		Notice:        field(f, pdufield.ShortMessage),
	}

	defer logRequests(r.Tid, p, err)

	log.WithFields(log.Fields{
		"fields": fmt.Sprintf("%#v", f),
	}).Debug("process")

	//_, err = resolveRec(string(f[pdufield.DestinationAddr].Bytes()), r)
	if err != nil {
		return
	}

	switch tlvfield(tlv, pdufield.SourcePort) {
	case "3": // subscription enabled
	case "4": // subscritpion disabled
	case "5": // charge notify. charged <sum> rub. Next date ...
	case "6": // block_subscribtion - unsubscribe
		unsubscribeAll(*r)
	case "7": // block subscriber
	case "9": // msisdn change
	}
}

func tlvfield(f pdufield.TLVMap, name pdufield.TLVType) string {
	v, ok := f[name]
	if !ok {
		return ""
	}
	return string(v.Bytes())
}

func field(f pdufield.Map, name pdufield.Name) string {
	v, ok := f[name]
	if !ok {
		return ""
	}
	return string(v.Bytes())
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
	serviceToken := dstAddress[0:4] // short number
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

func initTransceiver(conf config.SmppConfig, receiverFn func(p pdu.Body)) {
	if svc.transceiver == nil {
		reConnect(conf, receiverFn)
	}
	return
}

func reConnect(conf config.SmppConfig, receiverFn func(p pdu.Body)) {
	log.Info("smpp transceiver init...")
	svc.transceiver = &smpp_client.Transceiver{
		Addr:        conf.Addr,
		User:        conf.User,
		Passwd:      conf.Password,
		RespTimeout: time.Duration(conf.Timeout) * time.Second,
		Handler:     receiverFn,
	}

	for {
		connStatus := svc.transceiver.Bind()

		for c := range connStatus {
			if c == nil {
				log.WithFields(log.Fields{
					"status": c,
					"t":      svc.transceiver,
				}).Info("status is nil")
				break
			}
			if c.Status().String() != "Connected" {
				m.SMPPConnected.Set(0)
				m.SMPPReconnectCount.Inc()
				log.WithFields(log.Fields{
					"user":   conf.User,
					"status": c.Status().String(),
					"error":  "disconnected: " + c.Error().Error(),
				}).Error("smpp connect failed")
				log.WithField("reconnectDelay", conf.ReconnectDelay).Info("smpp reconnects...")
				time.Sleep(time.Duration(conf.ReconnectDelay) * time.Second)
			} else {
				log.WithFields(log.Fields{
					"status": c.Status().String(),
				}).Info("smpp connect ok")
				m.SMPPConnected.Set(1)
				m.SMPPReconnectCount.Set(0)
				break
			}
		}
	}

	log.Info("smpp transmitter init done")
}
