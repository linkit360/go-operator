package service

// only sends to yondu, async
import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"

	m "github.com/vostrok/operator/ph/yondu/src/metrics"
	"github.com/vostrok/utils/rec"
)

func processVerifyTransCode(deliveries <-chan amqp.Delivery) {

	for msg := range deliveries {
		var t rec.Record
		var err error
		var operatorErr error
		var code int
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
			}).Error("consume from " + svc.conf.Yondu.Queue.VerifyTransCode.Name)
			goto ack
		}

		t = e.EventData

		//<-svc.api.ThrottleMT
		code, err = strconv.Atoi(t.SMSText)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err.Error(),
				"msg":        "dropped",
				"code":       t.SMSText,
				"chargeBody": string(msg.Body),
			}).Error("code is incorrect " + svc.conf.Yondu.Queue.VerifyTransCode.Name)
			goto ack
		}

		yResp, operatorErr = svc.api.VerifyTransCode(t.Msisdn, code)
		logRequests("verifytranscode", t, yResp, begin, operatorErr)
		if err := svc.publishTransactionLog("sent_verify_transcode", t); err != nil {
			log.WithFields(log.Fields{
				"event": e.EventName,
				"tid":   t.Tid,
				"error": err.Error(),
			}).Error("sent to transaction log failed")
		} else {
			log.WithFields(log.Fields{
				"queue": svc.conf.Yondu.Queue.VerifyTransCode.Name,
				"event": e.EventName,
				"tid":   t.Tid,
			}).Info("success(sent to telco, sent to transaction log)")
		}
		if operatorErr != nil {
			log.WithFields(log.Fields{
				"rec":   fmt.Sprintf("%#v", t),
				"msg":   "requeue",
				"error": err.Error(),
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
