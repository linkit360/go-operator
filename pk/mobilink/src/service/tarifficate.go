package service

// get records from queue  *_requests
// send to operator charge request
// send result to another queue

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"

	m "github.com/vostrok/operator/pk/mobilink/src/metrics"
	rec "github.com/vostrok/utils/rec"
)

type EventNotifyUserActions struct {
	EventName string     `json:"event_name,omitempty"`
	EventData rec.Record `json:"event_data,omitempty"`
}

func processTarifficate(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		begin := time.Now()
		log.WithFields(log.Fields{
			"priority": msg.Priority,
			"body":     string(msg.Body),
		}).Debug("start process")

		var e EventNotifyUserActions
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			m.Dropped.Inc()

			log.WithFields(log.Fields{
				"error":   err.Error(),
				"msg":     "dropped",
				"request": string(msg.Body),
			}).Error("consume failed")
			goto ack
		}

		switch {
		case e.EventName == "charge":
			t := e.EventData

			var err error

			<-svc.api.ThrottleMT
			if err = svc.api.Tarifficate(&t); err != nil {
				log.WithFields(log.Fields{
					"error":  err.Error(),
					"action": "requeue",
				}).Error("can't process")

			nack:
				if err := msg.Nack(false, true); err != nil {
					log.WithFields(log.Fields{
						"tid":   t.Tid,
						"error": err.Error(),
					}).Error("cannot ack")
					time.Sleep(time.Second)
					goto nack
				}
				continue
			}

			if err := svc.publishResponse("operator_response", t); err != nil {
				m.Dropped.Inc()

				log.WithFields(log.Fields{
					"event": e.EventName,
					"tid":   t.Tid,
					"error": err.Error(),
				}).Error("charge publish")
			} else {
				log.WithFields(log.Fields{
					"event": e.EventName,
					"tid":   t.Tid,
					"paid":  t.Paid,
					"took":  time.Since(begin),
				}).Info("processed successfully")
			}
		default:
			m.Dropped.Inc()

			log.WithFields(log.Fields{
				"eventName": e.EventName,
				"data":      fmt.Sprintf("%#v", e.EventData),
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
