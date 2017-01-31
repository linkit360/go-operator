// can make MT requests (request to tarifficate a msisdn via Mobilink API
// it is assumed we could edit settings from admin interface in the future,
// so there is a setting field in operators table (it is not used now)
//
// Mobilink provides also SMS send interface which could be used outside the module
// unique token used to check statistics with Mobilink using transactions table
package mobilink

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	smpp_client "github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	"github.com/gin-gonic/gin"

	m "github.com/vostrok/operator/pk/mobilink/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
	"github.com/vostrok/utils/amqp"
	rec "github.com/vostrok/utils/rec"
)

func (mb *Mobilink) Tarifficate(record *rec.Record) error {
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

func (mb *Mobilink) balanceCheck(tid, msisdn string) (bool, error) {
	if !My(msisdn) {
		return false, nil
	}
	token := getToken(msisdn)
	now := time.Now().In(mb.location).Format("20060102T15:04:05-0700")
	requestBody := mb.conf.Connection.MT.CheckBalanceBody
	requestBody = strings.Replace(requestBody, "%msisdn%", msisdn[2:], 1)
	requestBody = strings.Replace(requestBody, "%token%", token, 1)
	requestBody = strings.Replace(requestBody, "%time%", now, 1)

	req, err := http.NewRequest("POST", mb.conf.Connection.MT.Url, strings.NewReader(requestBody))
	if err != nil {
		err = fmt.Errorf("http.NewRequest: %s", err.Error())
		return false, err
	}
	for k, v := range mb.conf.Connection.MT.Headers {
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
			"endpoint":        mb.conf.Connection.MT.Url,
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

	for _, v := range mb.conf.Connection.MT.PostPaidBodyContains {
		if strings.Contains(string(mobilinkResponse), v) {
			postPaid = true
			return true, nil
		}
	}
	return false, nil

}
func (mb *Mobilink) mt(r *rec.Record) error {
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
	requestBody := mb.conf.Connection.MT.TarifficateBody

	if mb.conf.AddBalanceServiceId > 0 && r.ServiceId == mb.conf.AddBalanceServiceId {
		requestBody = strings.Replace(requestBody, "%price%", strconv.Itoa(price), 1)
		log.WithFields(log.Fields{
			"tid":    tid,
			"msisdn": msisdn,
		}).Info("add balance")
	} else {
		requestBody = strings.Replace(requestBody, "%price%", "-"+strconv.Itoa(price), 1)
	}
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

	req, err := http.NewRequest("POST", mb.conf.Connection.MT.Url, strings.NewReader(requestBody))
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
	for k, v := range mb.conf.Connection.MT.Headers {
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
			"endpoint":        mb.conf.Connection.MT.Url,
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
		"endpoint":    mb.conf.Connection.MT.Url,
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
			ServiceId:        r.ServiceId,
			SubscriptionId:   r.SubscriptionId,
			CampaignId:       r.CampaignId,
			RequestBody:      strings.TrimSpace(requestBody),
			ResponseBody:     strings.TrimSpace(string(mobilinkResponse)),
			ResponseDecision: responseDecision,
			ResponseCode:     responseCode,
			SentAt:           time.Now().UTC(),
		}
		mb.publishTransactionLog(msg)

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
	for _, v = range mb.conf.Connection.MT.PaidBodyContains {
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

func (mb *Mobilink) SendSMS(tid, msisdn, msg string) error {
	m.SmsSuccess.Inc()
	shortMsg, err := mb.smpp.Submit(&smpp_client.ShortMessage{
		Src:      mb.conf.Connection.Smpp.ShortNumber,
		Dst:      "00" + msisdn[2:],
		Text:     pdutext.Raw(msg),
		Register: smpp_client.NoDeliveryReceipt,
	})

	if err == smpp_client.ErrNotConnected {
		m.SmsError.Inc()
		m.Errors.Inc()

		log.WithFields(log.Fields{
			"msisdn": msisdn,
			"msg":    msg,
			"tid":    tid,
			"error":  err.Error(),
		}).Error("counldn't send sms: service unavialable")
		return fmt.Errorf("smpp.Submit: %s", err.Error())
	}
	if err != nil {
		m.SmsError.Inc()
		m.Errors.Inc()

		log.WithFields(log.Fields{
			"msisdn": msisdn,
			"msg":    msg,
			"tid":    tid,
			"error":  err.Error(),
		}).Error("counldn't send sms: bad request")
		return fmt.Errorf("smpp.Submit: %s", err.Error())
	}

	log.WithFields(log.Fields{
		"msisdn": msisdn,
		"msg":    msg,
		"tid":    tid,
		"respid": shortMsg.RespID(),
	}).Info("sms sent")
	return nil
}
func getToken(msisdn string) string {
	return msisdn + time.Now().Format("20060102150405")[6:]
}
func AddMobilinkTestHandlers(r *gin.Engine) {
	rgMobilink := r.Group("/mobilink")
	rgMobilink.POST("/paid", paidHandler)
	rgMobilink.POST("/failed", failedHandler)
	rgMobilink.POST("/postpaid", postPaidHandler)
}
func paidHandler(c *gin.Context) {
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte(`<value><i4>0</i4></value>`))
}
func failedHandler(c *gin.Context) {
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte(`<value><i4>112</i4></value>`))
}
func postPaidHandler(c *gin.Context) {
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte(`<value><i4>11</i4></value>`))
}
func (mb *Mobilink) publishTransactionLog(data interface{}) error {
	event := amqp.EventNotify{
		EventName: "send_to_db",
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		m.Errors.Inc()
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	mb.notifier.Publish(amqp.AMQPMessage{mb.conf.TransactionLog.Queue, 0, body})
	return nil
}
