package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"

	m "github.com/vostrok/operator/th/qrtech/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	rec "github.com/vostrok/utils/rec"
)

// MT handler
func AddTestHandlers(r *gin.Engine) {
	//:= r.Group("/qr/mt")
}

func processCharge(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {
		var t rec.Record
		var err error
		var operatorErr error

		logCtx := log.WithFields(log.Fields{
			"q": svc.conf.QRTech.Queue.MT,
		})
		var e EventNotify
		if err = json.Unmarshal(msg.Body, &e); err != nil {
			m.Dropped.Inc()

			logCtx.WithFields(log.Fields{
				"error": err.Error(),
				"msg":   "dropped",
				"body":  string(msg.Body),
			}).Error("failed")
			goto ack
		}

		t = e.EventData
		logCtx = logCtx.WithFields(log.Fields{
			"tid": t.Tid,
		})

		<-svc.API.Throttle.MT
		operatorErr = svc.API.mt(t)
		if operatorErr != nil {
			logCtx.WithFields(log.Fields{
				"msg":   "requeue",
				"error": operatorErr.Error(),
			}).Error("can't process")
			msg.Nack(false, true)
			continue
		}
	ack:
		if err = msg.Ack(false); err != nil {
			logCtx.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("cannot ack")
			time.Sleep(time.Second)
			goto ack
		}

	}
}

func (qr *QRTech) mt(r rec.Record) (err error) {
	logCtx := log.WithFields(log.Fields{
		"q": svc.conf.QRTech.Queue.MT,
	})
	v := url.Values{}
	var resp *http.Response
	var qrTechResponse []byte
	defer func() {
		tl := transaction_log_service.OperatorTransactionLog{
			Tid:              r.Tid,
			Msisdn:           r.Msisdn,
			OperatorToken:    r.OperatorToken,
			OperatorCode:     r.OperatorCode,
			CountryCode:      r.CountryCode,
			Error:            r.OperatorErr,
			Price:            r.Price,
			ServiceId:        r.ServiceId,
			CampaignId:       r.CampaignId,
			RequestBody:      qr.conf.MT.APIUrl + "?" + v.Encode(),
			ResponseBody:     string(qrTechResponse),
			ResponseDecision: "",
			ResponseCode:     resp.StatusCode,
			SentAt:           r.SentAt,
			Notice:           "",
			Type:             "mt",
		}
		logResponse("mt", r, tl, err)
		if err = svc.publishTransactionLog(tl); err != nil {
			logCtx.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("sent to transaction log failed")
		}
	}()

	v.Add("username", qr.conf.MT.UserName)
	v.Add("serviceid", strconv.FormatInt(r.ServiceId, 10))
	v.Add("broadcastdate", time.Now().Format("20060102150405")[:8])
	v.Add("ctype", "2")                               // unicode type
	v.Add("message", strconv.QuoteToASCII(r.SMSText)) // when sees any non-ascii type, converts to unicode
	//v.Add("header", 1) // todo: mandatory?

	logCtx.WithFields(log.Fields{
		"url":    qr.conf.MT.APIUrl,
		"params": v.Encode(),
	}).Debug("call...")

	req, err := http.NewRequest("POST", qr.conf.MT.APIUrl, strings.NewReader(v.Encode()))
	if err != nil {
		err = fmt.Errorf("Cann't create request: %s", err.Error())
		return
	}
	req.Close = false

	resp, err = qr.client.Do(req)
	if err != nil {
		err = fmt.Errorf("Cann't make request: %s", err.Error())
		return
	}
	if resp.StatusCode > 220 {
		err = fmt.Errorf("qrTech resp status: %s", resp.Status)
		return
	}
	qrTechResponse, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("ioutil.ReadAll: %s", err.Error())
		return
	}
	defer resp.Body.Close()

	logCtx.WithFields(log.Fields{
		"response": string(qrTechResponse),
	}).Debug("got response")

	return
}
