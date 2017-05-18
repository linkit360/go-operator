package service

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"

	log "github.com/Sirupsen/logrus"
	smpp_client "github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"

	"github.com/linkit360/go-operator/ru/beeline/src/config"
	m "github.com/linkit360/go-operator/ru/beeline/src/metrics"
	"github.com/linkit360/go-utils/rec"
)

type Incoming struct {
	Tid          string    `json:"tid"`
	Seq          string    `json:"seq"`
	SentAt       time.Time `json:"sent_at"`
	SourcePort   uint16    `json:"source_port"`
	DstAddr      string    `json:"dst_addres"`
	ShortMessage string    `json:"short_message"`
	SourceAddr   string    `json:"source_addr"`
}

func pduHandler(p pdu.Body) {
	m.Incoming.Inc()

	f := p.Fields()
	h := p.Header()
	tlv := p.TLVFields()

	i := Incoming{
		Tid:          rec.GenerateTID(field(f, pdufield.SourceAddr)),
		Seq:          fmt.Sprintf("%v", h.Seq),
		SentAt:       time.Now().UTC(),
		SourcePort:   tlvfieldi(tlv, pdufield.SourcePort),
		DstAddr:      field(f, pdufield.DestinationAddr),
		ShortMessage: field(f, pdufield.ShortMessage),
		SourceAddr:   field(f, pdufield.SourceAddr),
	}

	fields := log.Fields{
		"id":            h.ID,
		"len":           h.Len,
		"seq":           h.Seq,
		"status":        h.Status,
		"tid":           i.Tid,
		"sent_at":       i.SentAt,
		"msisdn":        i.SourceAddr,
		"dst":           i.DstAddr,
		"source_port":   i.SourcePort,
		"short_message": i.ShortMessage,
	}
	log.WithFields(fields).Debug("pdu")
	svc.requestLog.WithFields(fields).Println(".")

	if len(i.DstAddr) < 4 {
		m.AbsentParameter.Inc()
		m.Errors.Inc()

		log.WithFields(log.Fields{
			"dstAddr": i.DstAddr,
			"error":   "addr is wrong",
		}).Error("cann't process")
		return
	}
	if err := publish(i); err != nil {
		m.Success.Inc()
	}
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
	log.WithFields(log.Fields{
		"utf8_valid": utf8.Valid(v.Bytes()),
		"raw":        fmt.Sprintf("%#v", v.Raw()),
		"string":     v.String(),
		"ascii":      strconv.QuoteToASCII(v.String()),
		"name":       fmt.Sprintf("%#v", v.Bytes()),
	}).Debug("bytes")
	return v.String()
}

func reConnectTranceiver(conf config.SmppConfig, receiverFn func(p pdu.Body)) {
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
