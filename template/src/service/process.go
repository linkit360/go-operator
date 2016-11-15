package service

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/expvar"
	"github.com/streadway/amqp"

	rec "github.com/vostrok/mt_manager/src/service/instance"
)

type Metrics struct {
	Dropped metrics.Counter
	Empty   metrics.Counter
}

func initMetrics() Metrics {
	return Metrics{
		Dropped: expvar.NewCounter("dropped"),
		Empty:   expvar.NewCounter("empty"),
	}
}

type EventNotifyUserActions struct {
	EventName string     `json:"event_name,omitempty"`
	EventData rec.Record `json:"event_data,omitempty"`
}

func process(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		log.WithFields(log.Fields{
			"body": string(msg.Body),
		}).Debug("start process")

		var e EventNotifyUserActions
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			svc.m.Dropped.Add(1)

			log.WithFields(log.Fields{
				"error":       err.Error(),
				"msg":         "dropped",
				"contentSent": string(msg.Body),
			}).Error("consume from " + svc.conf.server)
			msg.Ack(false)
			continue
		}

		if e.EventName == "charge" {
			t := e.EventData
			query := ""
			var err error

			// types of cann't process
			// it isn't types when operator or our error occurs
			if err != nil {
				msg.Nack(false, true)

				log.WithFields(log.Fields{
					"rec":   fmt.Sprintf("%#v", t),
					"query": query,
					"msg":   "requeue",
					"error": err.Error(),
				}).Error("can't process")
				continue
			}

			log.WithFields(log.Fields{
				"rec": t,
			}).Info("processed successfully")
		} else {
			svc.m.Dropped.Add(1)
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
