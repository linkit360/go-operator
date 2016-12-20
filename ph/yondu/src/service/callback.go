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

func processCallBack(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		t := rec.Record{}

		log.WithFields(log.Fields{
			"priority": msg.Priority,
			"body":     string(msg.Body),
		}).Debug("start process")

		var e EventNotifyResponse

		if err := json.Unmarshal(msg.Body, &e); err != nil {
			m.Dropped.Inc()

			log.WithFields(log.Fields{
				"error": err.Error(),
				"msg":   "dropped",
				"mo":    string(msg.Body),
			}).Error("consume from " + svc.conf.Yondu.Queue.CallBack.Name)
			goto ack
		}
		// prepare rec. record
		//yondyResponse := e.EventData
		// parse callback response
		// notify in yondu_responses

		if err := svc.publishTransactionLog("callback", t); err != nil {
			log.WithFields(log.Fields{
				"event": e.EventName,
				"mo":    msg.Body,
				"error": err.Error(),
			}).Error("sent to transaction log failed")
		nack:
			if err := msg.Nack(false, true); err != nil {
				log.WithFields(log.Fields{
					"mo":    msg.Body,
					"error": err.Error(),
				}).Error("cannot nack")
				time.Sleep(time.Second)
				goto nack
			}
			continue
		} else {
			log.WithFields(log.Fields{
				"queue": svc.conf.Yondu.Queue.CallBack.Name,
				"event": e.EventName,
				"tid":   t.Tid,
			}).Info("success (sent to transaction log)")
		}
	ack:
		if err := msg.Ack(false); err != nil {
			log.WithFields(log.Fields{
				"mo":    msg.Body,
				"error": err.Error(),
			}).Error("cannot ack")
			time.Sleep(time.Second)
			goto ack
		}

	}
}
