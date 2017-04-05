package service

import (
	"bytes"
	"encoding/binary"
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

	sourcePort := tlvfieldi(tlv, pdufield.SourcePort)
	dstAddr := field(f, pdufield.DestinationAddr)
	shortMessage := field(f, pdufield.ShortMessage)
	sourceAddr := field(f, pdufield.SourceAddr)

	r := &rec.Record{
		Tid:           rec.GenerateTID(),
		CountryCode:   svc.conf.Beeline.CountryCode,
		OperatorCode:  svc.conf.Beeline.MccMnc,
		Msisdn:        sourceAddr,
		OperatorToken: strconv.FormatUint(uint64(h.Seq), 10),
		Notice:        shortMessage + fmt.Sprintf(" source_port: %v", sourcePort),
	}
	logCtx := log.WithField("tid", r.Tid)

	fields := log.Fields{
		"tid":          r.Tid,
		"id":           h.ID,
		"len":          h.Len,
		"seq":          h.Seq,
		"status":       h.Status,
		"msisdn":       sourceAddr,
		"dst":          dstAddr,
		"source_port":  sourcePort,
		"shot_message": shortMessage,
	}
	log.WithFields(fields).Debug("pdu")
	defer logRequests(*r, fields, err)

	fieldsJSON, _ := json.Marshal(fields)

	_, err = resolveRec(dstAddr, r)
	tl := transaction_log_service.OperatorTransactionLog{
		Tid:           r.Tid,
		Msisdn:        r.Msisdn,
		OperatorToken: r.OperatorToken,
		OperatorCode:  r.OperatorCode,
		CountryCode:   r.CountryCode,
		Error:         r.OperatorErr,
		Price:         r.Price,
		ServiceId:     r.ServiceId,
		CampaignId:    r.CampaignId,
		RequestBody:   string(fieldsJSON),
		ResponseBody:  "",
		ResponseCode:  200,
		Notice:        r.Notice,
		Type:          "smpp",
	}
	if err != nil {
		svc.publishTransactionLog(tl)
		return
	}

	switch sourcePort {
	case 3:
		tl.ResponseDecision = "subscription enabled"
		logCtx.Debug(tl.ResponseDecision)
		newSubscription(*r)

	case 4:
		tl.ResponseDecision = "subscription disabled"
		r.SubscriptionStatus = "canceled"
		logCtx.Debug(tl.ResponseDecision)
		unsubscribe(*r)
	case 5:
		logCtx.Debug("charge notify")
		r.Paid = true
		r.SubscriptionStatus = "paid"
		r.Result = "paid"
		chargeNotify(*r)
	case 6:
		tl.ResponseDecision = "unsubscribe all"
		logCtx.Debug(tl.ResponseDecision)
		r.SubscriptionStatus = "canceled"
		unsubscribeAll(*r)
	case 1:
		tl.ResponseDecision = "unsubscribe all"
		logCtx.Debug(tl.ResponseDecision)
		r.SubscriptionStatus = "canceled"
		unsubscribeAll(*r)
	case 7:
		tl.ResponseDecision = "block"
		logCtx.Debug(tl.ResponseDecision)
	case 9:
		tl.ResponseDecision = "msisdn migration"
		logCtx.Debug(tl.ResponseDecision)
	}
	svc.publishTransactionLog(tl)
}

func tlvfieldi(f pdufield.TLVMap, name pdufield.TLVType) (field uint16) {
	v, ok := f[name]
	if !ok || v == nil {
		return
	}

	buf := bytes.NewReader(v.Bytes())
	err := binary.Read(buf, binary.BigEndian, &field)
	if err != nil {
		fmt.Println("binary.Read failed:", err)
		return
	}

	return
}

func field(f pdufield.Map, name pdufield.Name) string {
	v, ok := f[name]
	if !ok || v == nil {
		return ""
	}
	return v.String()
}

func resolveRec(dstAddress string, r *rec.Record) (service inmem_service.Service, err error) {
	logCtx := log.WithField("tid", r.Tid)

	if len(dstAddress) < 4 {
		m.AbsentParameter.Inc()
		m.Errors.Inc()

		err = fmt.Errorf("dstAddr required%s", "")

		logCtx.WithFields(log.Fields{
			"dstAddr": dstAddress,
			"error":   "dstAddress is wrong",
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

	r.Price = int(service.Price) * 100
	r.DelayHours = service.DelayHours
	r.PaidHours = service.PaidHours
	r.KeepDays = service.KeepDays
	r.Periodic = false
	return
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
