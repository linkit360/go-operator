package service

// get records from queue  *_requests
// send to operator charge request
// send result to another queue

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
			"priority": msg.Priority,
			"body":     string(msg.Body),
		}).Debug("start process")

		var e EventNotifyUserActions
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			svc.m.Dropped.Inc()

			log.WithFields(log.Fields{
				"error":       err.Error(),
				"msg":         "dropped",
				"contentSent": string(msg.Body),
			}).Error("consume from " + svc.conf.queues.Requests)
			msg.Ack(false)
			continue
		}

		switch {
		case e.EventName == "charge":
			t := e.EventData

			var err error

			<-svc.api.ThrottleMT
			if err = svc.api.Tarifficate(&t); err != nil {
				msg.Nack(false, true)
				log.WithFields(log.Fields{
					"rec":   fmt.Sprintf("%#v", t),
					"msg":   "requeue",
					"error": err.Error(),
				}).Error("can't process")
				continue
			}

			if err := svc.publishResponse("operator_response", t); err != nil {
				log.WithFields(log.Fields{
					"queue": svc.conf.queues.Requests,
					"event": e.EventName,
					"tid":   t.Tid,
					"error": err.Error(),
				}).Error("charge request error")

			} else {
				log.WithFields(log.Fields{
					"queue": svc.conf.queues.Requests,
					"event": e.EventName,
					"tid":   t.Tid,
				}).Info("processed successfully")
			}
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
