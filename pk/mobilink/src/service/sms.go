package service

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"

	m "github.com/linkit360/go-operator/pk/mobilink/src/metrics"
)

// get records to send sms from queue *_sms_requests
// send to operator
// send response to another queue
func processSMS(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		log.WithFields(log.Fields{
			"body": string(msg.Body),
		}).Debug("start process")

		var e EventNotifyUserActions
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			m.Dropped.Inc()

			log.WithFields(log.Fields{
				"error":       err.Error(),
				"msg":         "dropped",
				"sms_request": string(msg.Body),
			}).Error("consume failed")
			goto ack
		}

		switch {
		case e.EventName == "send_sms":
			t := e.EventData
			if err := svc.api.SendSMS(t.Tid, t.Msisdn, t.SMSText); err != nil {
				msg.Ack(false)
				log.WithFields(log.Fields{
					"rec":   fmt.Sprintf("%#v", t),
					"msg":   "requeue",
					"error": err.Error(),
				}).Error("can't process")
				t.OperatorErr = err.Error()
			}

			if err := svc.publishSMSResponse("sms_response", t); err != nil {
				log.WithFields(log.Fields{
					"event": e.EventName,
					"tid":   t.Tid,
					"error": err.Error(),
				}).Error("send sms error")
			} else {
				log.WithFields(log.Fields{
					"event": e.EventName,
					"tid":   t.Tid,
				}).Info("processed successfully")

			}
		default:
			m.Dropped.Inc()

			log.WithFields(log.Fields{
				"eventName": e.EventName,
				"msg":       "dropped",
			}).Error("unknown event name")
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
