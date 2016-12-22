package service

// only sends to yondu, async
import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"

	m "github.com/vostrok/operator/ph/yondu/src/metrics"
	"github.com/vostrok/utils/rec"
)

type EventNotifyResponse struct {
	EventName string                `json:"event_name,omitempty"`
	EventData YonduResponseExtended `json:"event_data,omitempty"`
}

func processCharge(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		var t rec.Record
		var operatorErr error
		var err error
		var amount string
		var ok bool
		var yResp YonduResponseExtended

		begin := time.Now()

		log.WithFields(log.Fields{
			"priority": msg.Priority,
			"body":     string(msg.Body),
		}).Debug("start process")

		var e EventNotify
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			m.Dropped.Inc()

			log.WithFields(log.Fields{
				"error":      err.Error(),
				"msg":        "dropped",
				"chargeBody": string(msg.Body),
			}).Error("consume from " + svc.conf.Yondu.Queue.Charge.Name)
			goto ack
		}

		t = e.EventData

		//<-svc.api.ThrottleMT
		amount, ok = svc.conf.Yondu.Tariffs[t.Price]
		if !ok {
			m.Dropped.Inc()

			err = fmt.Errorf("Unknown price to Yondu: %v", t.Price)
			log.WithFields(log.Fields{
				"error":      err.Error(),
				"msg":        "dropped",
				"chargeBody": string(msg.Body),
			}).Error("wrong price")
			goto ack
		}
		yResp, operatorErr = svc.api.Charge(t.Msisdn, amount)
		logRequests("charge", t, yResp, begin, operatorErr)
		if err := svc.publishTransactionLog("charge_request", yResp, t); err != nil {
			log.WithFields(log.Fields{
				"event": e.EventName,
				"tid":   t.Tid,
				"error": err.Error(),
			}).Error("sent to transaction log failed")
		} else {
			log.WithFields(log.Fields{
				"queue": svc.conf.Yondu.Queue.Charge.Name,
				"event": e.EventName,
				"tid":   t.Tid,
			}).Info("success(sent to telco, sent to transaction log)")
		}
		if operatorErr != nil {
			log.WithFields(log.Fields{
				"rec":   fmt.Sprintf("%#v", t),
				"msg":   "requeue",
				"error": operatorErr.Error(),
			}).Error("can't process")

			msg.Nack(false, true)
			continue
		}
	ack:
		if err := msg.Ack(false); err != nil {
			log.WithFields(log.Fields{
				"tid":   e.EventData.Tid,
				"error": err.Error(),
			}).Error("cannot ack")
			time.Sleep(time.Second)
			goto ack
		}

	}
}
