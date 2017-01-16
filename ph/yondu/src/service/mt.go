package service

// only sends to yondu, async
import (
	"encoding/json"
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
		var e EventNotify
		if err = json.Unmarshal(msg.Body, &e); err != nil {
			m.Dropped.Inc()

			logCtx.WithFields(log.Fields{
				"error": err.Error(),
				"msg":   "dropped",
				"body":  string(msg.Body),
			}).Error("failed")
			goto ack
		}

		t = e.EventData
		logCtx = logCtx.WithFields(log.Fields{
			"tid": t.Tid,
		})

		if t.SMSText == "" {
			m.Dropped.Inc()

			logCtx.WithFields(log.Fields{
				"msg": "dropped",
			}).Error("empty text")
			goto ack
		}
		<-svc.YonduAPI.Throttle.MT
		yResp, operatorErr = svc.YonduAPI.MT(t.Tid, t.Msisdn, t.SMSText)
		logRequests("mt", t, yResp, begin, operatorErr)
		//if err = svc.publishTransactionLog("mt", yResp, t); err != nil {
		//	logCtx.WithFields(log.Fields{
		//		"event": e.EventName,
		//		"error": err.Error(),
		//	}).Error("sent to transaction log failed")
		//}
		//if operatorErr != nil {
		//	logCtx.WithFields(log.Fields{
		//		"msg":   "requeue",
		//		"error": operatorErr.Error(),
		//	}).Error("can't process")
		//	msg.Nack(false, true)
		//	continue
		//}
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
