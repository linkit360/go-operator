package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	m "github.com/vostrok/operator/ru/beeline/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	rec "github.com/vostrok/utils/rec"
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

	r := rec.Record{
		Tid:           rec.GenerateTID(),
		CountryCode:   svc.conf.Beeline.CountryCode,
		OperatorCode:  svc.conf.Beeline.MccMnc,
		Msisdn:        f[pdufield.SourceAddr],
		OperatorToken: h.Seq,
		Notice:        f[pdufield.ShortMessage],
	}
	defer logRequests(r.Tid, p, err)

	if err := getCampaign(f[pdufield.DestinationAddr], r); err != nil {
		return
	}
	tlv := p.TLVFields()

	switch string(tlv[pdufield.SourcePort].Bytes()) {
	case "8":

	}
}

func getCampaign(dstAddress string, r *rec.Record) (err error) {
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

	res, err := inmem_client.GetCampaignByKeyWord(serviceToken)
	if err != nil {
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{
			"serviceToken": serviceToken,
		}).Error("cannot find campaign by serviceToken")

		return
	}

	r.CampaignId = res.Id
	r.ServiceId = res.ServiceId
	service, err := inmem_client.GetServiceById(res.ServiceId)
	if err != nil {
		m.Errors.Inc()

		logCtx.WithFields(log.Fields{
			"service_id": res.ServiceId,
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

func (svc *Service) pduBody(p pdu.Body) string {
	msg := &SmppMessage{
		Fields:    make(map[string]string, len(p.FieldList())),
		TlvFields: make(map[pdufield.TLVType]string, len(p.TLVFields())),
	}
	f := p.Fields()
	for l := range p.FieldList() {
		msg.Fields[l] = f[l].String()
	}
	for tlvType, v := range p.TLVFields() {
		msg.TlvFields[tlvType] = string(v.Bytes())
	}
	msg.Headers = p.Header()

	data, err := json.Marshal(msg)
	if err != nil {
		log.WithField("error", err.Error()).Error("cannot marshal body")
		return ""
	}
	return string(data)
}
func (svc *Service) transactionLog(p pdu.Body, r rec.Record) {
	sentAt := time.Now().UTC()
	f := p.Fields()

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
		RequestBody:      svc.pduBody(p),
		ResponseBody:     "",
		ResponseDecision: "",
		ResponseCode:     200,
		SentAt:           sentAt,
		Notice:           f[pdufield.ShortMessage].String(),
		Type:             "smpp",
	}
	svc.publishTransactionLog(tl)
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
					"status": c.Status().String(),
					"error":  "disconnected:" + c.Status().String(),
				}).Error("smpp connect failed")
			} else {
				log.WithFields(log.Fields{
					"status": c.Status().String(),
				}).Info("smpp moblink connect ok")
				m.SMPPConnected.Set(1)
			}
		}
	}()
	log.Info("smpp transmitter init done")
	return smppTransceiver
}
