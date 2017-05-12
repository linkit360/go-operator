package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"

	m "github.com/linkit360/go-operator/pk/mobilink/src/metrics"
	transaction_log_service "github.com/linkit360/go-qlistener/src/service"
)

// get records to send sms from queue *_sms_requests
// send to operator
// send response to another queue
func processSMS(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		log.WithFields(log.Fields{
			"body": string(msg.Body),
		}).Debug("start process")

		var e EventNotify
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
			if !svc.conf.mb.Content.Enabled {
				m.Dropped.Inc()

				log.WithFields(log.Fields{
					"msg":         "dropped",
					"sms_request": string(msg.Body),
				}).Info("sms send disabled")
				goto ack
			}
			if e.EventData.SMSText == "" {
				m.Dropped.Inc()

				log.WithFields(log.Fields{
					"msg":         "dropped",
					"sms_request": string(msg.Body),
				}).Info("sms text is empty")
				goto ack
			}

			t := e.EventData
			tl, smsErr := svc.api.SendSMS(t.Tid, t.Msisdn, t.SMSText)
			if smsErr != nil {
				t.OperatorErr = smsErr.Error()
			}

			if err := svc.notifyTransactionLog("sms", transaction_log_service.OperatorTransactionLog{
				Tid:              t.Tid,
				Msisdn:           t.Msisdn,
				OperatorCode:     t.OperatorCode,
				CountryCode:      t.CountryCode,
				Error:            t.OperatorErr,
				Price:            t.Price,
				ServiceId:        t.ServiceId,
				SubscriptionId:   t.SubscriptionId,
				CampaignId:       t.CampaignId,
				RequestBody:      tl.RequestBody,
				ResponseBody:     tl.ResponseBody,
				ResponseDecision: tl.ResponseDecision,
				ResponseCode:     tl.ResponseCode,
				Type:             tl.Type,
			}); err != nil {
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
			if smsErr != nil {
				log.WithFields(log.Fields{
					"rec":   fmt.Sprintf("%#v", t),
					"error": smsErr.Error(),
				}).Error("requeue")
				time.Sleep(time.Second)
				msg.Nack(false, true)
				continue
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

func (mb *mobilink) SendSMS(tid, msisdn, msg string) (tl transaction_log_service.OperatorTransactionLog, err error) {
	tl.Type = "sms"
	m.SmsSuccess.Inc()

	v := url.Values{}
	v.Add("username", mb.conf.Content.User)
	v.Add("password", mb.conf.Content.Password)
	v.Add("from", mb.conf.Content.From)
	v.Add("smsc", mb.conf.Content.SMSC)
	v.Add("to", msisdn)
	v.Add("text", msg)

	endpoint := mb.conf.Content.Endpoint + "?" + v.Encode()
	logCtx := log.WithFields(log.Fields{
		"msisdn": msisdn,
		"text":   msg,
		"tid":    tid,
		"url":    endpoint,
	})
	tl.RequestBody = endpoint

	var req *http.Request
	req, err = http.NewRequest("GET", endpoint, nil)
	if err != nil {
		err = fmt.Errorf("http.NewRequest: %s", err.Error())
		logCtx.Error(err.Error())
		tl.Error = err.Error()
		tl.ResponseDecision = "failed"
		return
	}
	req.Close = false

	var resp *http.Response
	resp, err = mb.client.Do(req)
	if err != nil {
		m.SmsError.Inc()
		m.Errors.Inc()

		err = fmt.Errorf("GET client.Do: %s", err.Error())
		logCtx.Error(err.Error())
		tl.Error = err.Error()
		tl.ResponseDecision = "failed"
		return
	}
	tl.ResponseCode = resp.StatusCode

	if resp.StatusCode == 200 ||
		resp.StatusCode == 201 ||
		resp.StatusCode == 202 {
		var bodyText []byte
		bodyText, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			logCtx.WithFields(log.Fields{
				"err":            err.Error(),
				"respStatusCode": resp.StatusCode,
			}).Error("cannot read resp.Body for sms")
		} else {
			tl.ResponseBody = string(bodyText)
		}

		logCtx.WithFields(log.Fields{
			"body":           string(bodyText),
			"respStatusCode": resp.StatusCode,
		}).Debug("sms sent")

		tl.ResponseDecision = "sent"
		return
	}
	m.SmsError.Inc()
	m.Errors.Inc()

	err = fmt.Errorf("client.Do status code: %d", resp.StatusCode)
	logCtx.Error(err.Error())
	tl.Error = err.Error()
	tl.ResponseDecision = "failed"
	return
}
