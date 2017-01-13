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
		logCtx := log.WithFields(log.Fields{
			"q": svc.conf.Yondu.Queue.Charge.Name,
		})

		logCtx.WithFields(log.Fields{
			"priority": msg.Priority,
			"body":     string(msg.Body),
		}).Debug("start process")

		var e EventNotify
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			m.Dropped.Inc()

			logCtx.WithFields(log.Fields{
				"error": err.Error(),
				"msg":   "dropped",
				"body":  string(msg.Body),
			}).Error("consume")
			goto ack
		}

		t = e.EventData
		logCtx = logCtx.WithFields(log.Fields{
			"tid": t.Tid,
		})
		//<-svc.api.ThrottleMT
		amount, ok = svc.conf.Yondu.Tariffs[t.Price]
		if !ok || amount == "" {
			m.Dropped.Inc()

			err = fmt.Errorf("Unknown to Yondu: %v, %s", t.Price, amount)
			logCtx.WithFields(log.Fields{
				"error": err.Error(),
				"msg":   "dropped",
			}).Error("wrong price or empty amount")
			goto ack
		}
		<-svc.YonduAPI.Throttle.Charge
		yResp, operatorErr = svc.YonduAPI.Charge(t.Tid, t.Msisdn, amount)
		logRequests("charge", t, yResp, begin, operatorErr)
		if err = svc.publishTransactionLog("charge", yResp, t); err != nil {
			logCtx.WithFields(log.Fields{
				"event": e.EventName,
				"error": err.Error(),
			}).Error("sent to transaction log failed")
		}
		if operatorErr != nil {
			logCtx.WithFields(log.Fields{
				"rec":   fmt.Sprintf("%#v", t),
				"msg":   "requeue",
				"error": operatorErr.Error(),
			}).Error("can't process")

			msg.Nack(false, true)
			continue
		}
	ack:
		if err = msg.Ack(false); err != nil {
			logCtx.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("cannot ack")
			time.Sleep(time.Second)
			goto ack
		}

	}
}
