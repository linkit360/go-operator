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

func processSentConsent(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		var t rec.Record
		var err error
		var operatorErr error
		var ok bool
		var amount string
		var yResp YonduResponseExtended
		begin := time.Now()

		logCtx := log.WithFields(log.Fields{
			"q": svc.conf.Yondu.Queue.SendConsent.Name,
		})
		var e EventNotify
		if err = json.Unmarshal(msg.Body, &e); err != nil {
			m.Dropped.Inc()

			logCtx.WithFields(log.Fields{
				"error": err.Error(),
				"msg":   "dropped",
				"body":  string(msg.Body),
			}).Error("consume ")
			goto ack
		}

		t = e.EventData
		logCtx = logCtx.WithFields(log.Fields{
			"tid": t.Tid,
		})
		amount, ok = svc.conf.Yondu.Tariffs[t.Price]
		if !ok || amount == "" {
			m.Dropped.Inc()

			err = fmt.Errorf("Unknown to Yondu: %v, %s", t.Price, amount)
			logCtx.WithFields(log.Fields{
				"amount": amount,
				"price":  t.Price,
				"error":  err.Error(),
				"msg":    "dropped",
			}).Error("wrong price or empty amount")
			goto ack
		}
		<-svc.YonduAPI.Throttle.Consent
		yResp, operatorErr = svc.YonduAPI.SendConsent(t.Tid, t.Msisdn, amount)
		logRequests("sentconsent", t, yResp, begin, operatorErr)
		if err = svc.publishTransactionLog("consent", yResp, t); err != nil {
			logCtx.WithFields(log.Fields{
				"event": e.EventName,
				"error": err.Error(),
			}).Error("sent to transaction log failed")
		}

		if operatorErr != nil {
			logCtx.WithFields(log.Fields{
				"msg":   "requeue",
				"error": operatorErr.Error(),
			}).Error("can't process")

		nack:
			if err := msg.Nack(false, true); err != nil {
				logCtx.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("cannot nack")
				time.Sleep(time.Second)
				goto nack
			}
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
