package service

// get records from queue  *_requests
// send to operator charge request
// send result to another queue

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	m "github.com/linkit360/go-operator/pk/mobilink/src/metrics"
	transaction_log_service "github.com/linkit360/go-qlistener/src/service"
	rec "github.com/linkit360/go-utils/rec"
)

type EventNotify struct {
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

		var e EventNotify
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
				msg.Nack(false, true)
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

func (mb *mobilink) Tarifficate(record *rec.Record) error {
	log.WithFields(log.Fields{
		"rec": record,
	}).Info("start processing")
	var err error

	postPaid, err := mb.balanceCheck(record.Tid, record.Msisdn)
	if err != nil {
		m.Errors.Inc()
		m.BalanceCheckErrors.Inc()
		record.OperatorErr = err.Error()
		return err
	} else {
		m.BalanceCheckSuccess.Inc()
	}
	if postPaid {
		record.SubscriptionStatus = "postpaid"
		return nil
	}
	if err = mb.mt(record); err != nil {
		m.Errors.Inc()
		m.ChargeErrors.Inc()
		return err
	}
	return nil
}

func (mb *mobilink) balanceCheck(tid, msisdn string) (bool, error) {
	if !My(msisdn) {
		return false, nil
	}
	token := mb.getToken(msisdn)
	now := time.Now().In(mb.location).Format("20060102T15:04:05-0700")
	requestBody := mb.conf.MT.CheckBalanceBody
	requestBody = strings.Replace(requestBody, "%msisdn%", msisdn[2:], 1)
	requestBody = strings.Replace(requestBody, "%token%", token, 1)
	requestBody = strings.Replace(requestBody, "%time%", now, 1)

	req, err := http.NewRequest("POST", mb.conf.MT.Url, strings.NewReader(requestBody))
	if err != nil {
		err = fmt.Errorf("http.NewRequest: %s", err.Error())
		return false, err
	}
	for k, v := range mb.conf.MT.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Length", strconv.Itoa(len(requestBody)))
	req.Close = false

	var mobilinkResponse []byte
	postPaid := false
	begin := time.Now()
	defer func() {
		fields := log.Fields{
			"tid":             tid,
			"postPaid":        postPaid,
			"msisdn":          msisdn,
			"token":           token,
			"method":          "POST",
			"endpoint":        mb.conf.MT.Url,
			"headers":         fmt.Sprintf("%#v", req.Header),
			"reqeustBody":     requestBody,
			"requestResponse": string(mobilinkResponse),
			"took":            time.Since(begin),
		}
		if err != nil {
			fields["error"] = err.Error()
		}
		log.WithFields(fields).Info("mobilink check balance")
	}()

	resp, err := mb.client.Do(req)
	if err != nil {
		err = fmt.Errorf("client.Do: %s", err.Error())
		m.BalanceCheckDuration.Observe(time.Since(begin).Seconds())
		return false, err
	}
	m.BalanceCheckDuration.Observe(time.Since(begin).Seconds())

	mobilinkResponse, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("ioutil.ReadAll: %s", err.Error())
		return false, err
	}
	defer resp.Body.Close()

	for _, v := range mb.conf.MT.PostPaidBodyContains {
		if strings.Contains(string(mobilinkResponse), v) {
			postPaid = true
			return true, nil
		}
	}
	return false, nil
}

func (mb *mobilink) mt(r *rec.Record) error {
	msisdn := r.Msisdn
	tid := r.Tid
	price := r.Price

	if !My(msisdn) {
		log.WithFields(log.Fields{
			"msisdn": msisdn,
			"tid":    tid,
		}).Debug("is not mobilink")
		return nil
	}
	r.Paid = false
	r.OperatorToken = msisdn + time.Now().Format("20060102150405")[6:]
	now := time.Now().In(mb.location).Format("20060102T15:04:05-0700")

	requestBody := mb.conf.MT.TarifficateBody
	requestBody = strings.Replace(requestBody, "%price%", "-"+strconv.Itoa(price), 1)
	requestBody = strings.Replace(requestBody, "%msisdn%", msisdn[2:], 1)
	requestBody = strings.Replace(requestBody, "%token%", r.OperatorToken, 1)
	requestBody = strings.Replace(requestBody, "%time%", now, 1)

	log.WithFields(log.Fields{
		"token":  r.OperatorToken,
		"tid":    tid,
		"msisdn": msisdn,
		"price":  price,
		"time":   now,
	}).Debug("prepared for telco req")

	req, err := http.NewRequest("POST", mb.conf.MT.Url, strings.NewReader(requestBody))
	if err != nil {
		m.Errors.Inc()
		err = fmt.Errorf("http.NewRequest: %s", err.Error())

		log.WithFields(log.Fields{
			"token":  r.OperatorToken,
			"tid":    tid,
			"msisdn": msisdn,
			"time":   now,
			"error":  err.Error(),
		}).Error("create POST req to mobilink")

		return err
	}
	for k, v := range mb.conf.MT.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Length", strconv.Itoa(len(requestBody)))
	req.Close = false

	var responseCode int
	// transaction log for internal logging
	var mobilinkResponse []byte
	begin := time.Now()
	defer func() {
		fields := log.Fields{
			"token":           r.OperatorToken,
			"tid":             tid,
			"msisdn":          msisdn,
			"endpoint":        mb.conf.MT.Url,
			"headers":         fmt.Sprintf("%#v", req.Header),
			"reqeustBody":     requestBody,
			"requestResponse": string(mobilinkResponse),
			"took":            time.Since(begin),
		}
		if err != nil {
			fields["error"] = err.Error()
		}
		log.WithFields(fields).Info("mobilink")
	}()

	// separate transaction for mobilink
	// 1 - request body
	mb.requestLog.WithFields(log.Fields{
		"token":       r.OperatorToken,
		"tid":         tid,
		"msisdn":      msisdn,
		"endpoint":    mb.conf.MT.Url,
		"headers":     fmt.Sprintf("%#v", req.Header),
		"reqeustBody": strings.TrimSpace(requestBody),
	}).Info("mobilink request")
	defer func() {
		// separate transaction for mobilink
		// 2 - response body
		fields := log.Fields{
			"token":           r.OperatorToken,
			"tid":             tid,
			"msisdn":          msisdn,
			"requestResponse": strings.TrimSpace(string(mobilinkResponse)),
			"took":            time.Since(begin),
		}
		errStr := ""
		if err != nil {
			errStr = err.Error()
			fields["error"] = errStr
		}
		mb.responseLog.WithFields(fields).Println("mobilink response")

		var responseDecision string
		if r.Paid {
			responseDecision = "paid"
		} else {
			responseDecision = "failed"
		}
		msg := transaction_log_service.OperatorTransactionLog{
			Tid:              r.Tid,
			Msisdn:           r.Msisdn,
			OperatorToken:    r.OperatorToken,
			OperatorCode:     r.OperatorCode,
			CountryCode:      r.CountryCode,
			Error:            errStr,
			Price:            r.Price,
			ServiceCode:      r.ServiceCode,
			SubscriptionId:   r.SubscriptionId,
			CampaignCode:     r.CampaignId,
			RequestBody:      strings.TrimSpace(requestBody),
			ResponseBody:     strings.TrimSpace(string(mobilinkResponse)),
			ResponseDecision: responseDecision,
			ResponseCode:     responseCode,
			Type:             "charge",
		}
		svc.notifyTransactionLog("charge", msg)
	}()

	resp, err := mb.client.Do(req)
	if err != nil {
		r.OperatorErr = err.Error()

		err = fmt.Errorf("client.Do: %s", err.Error())
		log.WithFields(log.Fields{
			"error":  err,
			"token":  r.OperatorToken,
			"tid":    tid,
			"msisdn": msisdn,
			"time":   now,
		}).Error("do request to mobilink")
		m.ChargeDuration.Observe(time.Since(begin).Seconds())
		return nil
	}
	m.ChargeDuration.Observe(time.Since(begin).Seconds())

	responseCode = resp.StatusCode
	mobilinkResponse, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		r.OperatorErr = err.Error()

		err = fmt.Errorf("ioutil.ReadAll: %s", err.Error())
		log.WithFields(log.Fields{
			"token":  r.OperatorToken,
			"tid":    tid,
			"msisdn": msisdn,
			"time":   now,
			"error":  err,
		}).Error("get raw body of mobilink response")
		return err
	}
	defer resp.Body.Close()

	var v string
	for _, v = range mb.conf.MT.PaidBodyContains {
		if strings.Contains(string(mobilinkResponse), v) {
			m.SinceSuccessPaid.Set(.0)
			m.ChargeSuccess.Inc()
			log.WithFields(log.Fields{
				"msisdn": msisdn,
				"token":  r.OperatorToken,
				"tid":    tid,
				"price":  price,
			}).Info("charged")
			r.Paid = true
			return nil
		}
	}
	log.WithFields(log.Fields{
		"msisdn": msisdn,
		"tid":    tid,
		"price":  price,
	}).Info("charge has failed")

	return nil
}

func (mb *mobilink) getToken(msisdn string) string {
	return msisdn + time.Now().Format("20060102150405")[6:]
}
