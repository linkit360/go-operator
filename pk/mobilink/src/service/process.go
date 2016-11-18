package service

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"

	rec "github.com/vostrok/utils/rec"
)

type EventNotifyUserActions struct {
	EventName string     `json:"event_name,omitempty"`
	EventData rec.Record `json:"event_data,omitempty"`
}

func processTarifficate(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		log.WithFields(log.Fields{
			"body": string(msg.Body),
		}).Debug("start process")

		var e EventNotifyUserActions
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			svc.m.Dropped.Inc()

			log.WithFields(log.Fields{
				"error":       err.Error(),
				"msg":         "dropped",
				"contentSent": string(msg.Body),
			}).Error("consume from " + svc.conf.queues.In)
			msg.Ack(false)
			continue
		}

		switch {
		case e.EventName == "send_sms":
			t := e.EventData
			_ = svc.api.SendSMS(t.Tid, t.Msisdn, t.SMSText)
			msg.Ack(false)

		case e.EventName == "charge":
			t := e.EventData

			var err error

			<-svc.api.ThrottleMT
			svc.api.Tarifficate(&t)

			// types of cann't process
			// it isn't types when operator or our error occurs
			if err != nil {
				msg.Nack(false, true)

				log.WithFields(log.Fields{
					"rec":   fmt.Sprintf("%#v", t),
					"msg":   "requeue",
					"error": err.Error(),
				}).Error("can't process")
				continue
			}

			log.WithFields(log.Fields{
				"rec": t,
			}).Info("processed successfully")
		default:
			svc.m.Dropped.Inc()
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
