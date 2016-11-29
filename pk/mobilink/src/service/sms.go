package service

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"

	m "github.com/vostrok/operator/pk/mobilink/src/metrics"
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
				"contentSent": string(msg.Body),
			}).Error("consume from " + svc.conf.queues.SMSRequest)
			msg.Ack(false)
			continue
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
					"queue": svc.conf.queues.SMSRequest,
					"event": e.EventName,
					"tid":   t.Tid,
					"error": err.Error(),
				}).Error("send sms error")

			} else {
				log.WithFields(log.Fields{
					"queue": svc.conf.queues.SMSRequest,
					"event": e.EventName,
					"tid":   t.Tid,
				}).Info("processed successfully")

			}
		default:
			m.Dropped.Inc()
			msg.Ack(false)
			log.WithFields(log.Fields{
				"eventName": e.EventName,
				"msg":       "dropped",
			}).Error("unknown event name")
			continue
		}

		msg.Ack(false)
	}
}
