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

func processMT(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		var t rec.Record
		var err error
		var operatorErr error
		var yResp YonduResponseExtended
		begin := time.Now()

		logCtx := log.WithFields(log.Fields{
			"q": svc.conf.Yondu.Queue.MT.Name,
		})
		logCtx.WithFields(log.Fields{
			"priority": msg.Priority,
			"body":     string(msg.Body),
		}).Debug("start process")

		var e EventNotify
		if err = json.Unmarshal(msg.Body, &e); err != nil {
			m.Dropped.Inc()

			logCtx.WithFields(log.Fields{
				"error": err.Error(),
				"msg":   "dropped",
				"body":  string(msg.Body),
			}).Error("consume")
			goto ack
		}
		t = e.EventData

		//<-svc.api.ThrottleMT
		yResp, operatorErr = svc.YonduAPI.MT(t.Msisdn, t.SMSText)
		logRequests("mt", t, yResp, begin, operatorErr)
		if err = svc.publishTransactionLog("mt", yResp, t); err != nil {
			logCtx.WithFields(log.Fields{
				"event": e.EventName,
				"tid":   t.Tid,
				"error": err.Error(),
			}).Error("sent to transaction log failed")
		} else {
			logCtx.WithFields(log.Fields{
				"event": e.EventName,
			}).Info("success(sent to telco, sent to transaction log)")
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
