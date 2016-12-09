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

func processCharge(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		var t rec.Record
		var operatorErr error
		var amount string
		var yResp YonduResponse
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
		yResp, operatorErr = svc.api.Charge(t.Msisdn, amount)
		logResponse("charge", t, yResp, begin, operatorErr)
		if err := svc.publishTransactionLog("sent_charge_request", t); err != nil {
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
		nack:
			if err := msg.Nack(false, true); err != nil {
				log.WithFields(log.Fields{
					"tid":   e.EventData.Tid,
					"error": err.Error(),
				}).Error("cannot nack")
				time.Sleep(time.Second)
				goto nack
			}
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
