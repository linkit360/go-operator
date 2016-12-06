package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	//"github.com/gin-gonic/gin"

	m "github.com/vostrok/operator/ph/yondu/src/metrics"
	//transaction_log_service "github.com/vostrok/qlistener/src/service"
	"github.com/vostrok/utils/amqp"
	//rec "github.com/vostrok/utils/rec"
)

func (y *Yondu) publishTransactionLog(data interface{}) error {
	event := amqp.EventNotify{
		EventName: "send_to_db",
		EventData: data,
	}
	body, err := json.Marshal(event)
	if err != nil {
		m.Errors.Inc()
		return fmt.Errorf("json.Marshal: %s", err.Error())
	}
	y.notifier.Publish(amqp.AMQPMessage{y.conf.TransactionLog.Queue, 0, body})
	return nil
}

type SendConsentResponse struct {
	Response struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"response"`
}

//API URL: {URL}/m360api/v1/consent/{msisdn}/{amount}
//Description: To send transaction code as consent before charging the subscriber
//Method: GET
//Headers: Authorization: Bearer {token}
//Required parameters: msisdn, amount
//Sample Response: {"response":{"message":"verification sent","code":"2001"}}
//Sample Request: {URL}/m360api/v1/consent/9171234567/P1
func (y *Yondu) SendConsent(msisdn, amount string) (err error) {
	return y.call(y.conf.APIUrl+"/consent/"+msisdn+"/"+amount, 2001)
}

//API URL: {URL}/m360api/v1/verify/{msisdn}/{code}
//Description: To verify transaction code sent to subscriber if input is not through SMS
//Method: GET
//Headers: Authorization: Bearer {token}
//Required parameters: msisdn, code
//Sample Response: {"response":{"message":"verification successful","code":"2003"}}
// Sample Request: {URL}/m360api/v1/verify/9171234567/123456
func (y *Yondu) VerifyTransCode(msisdn string, code int) (err error) {
	return y.call(y.conf.APIUrl+"/verify/"+msisdn+"/"+code, 2003)
}

//API URL: {URL}/m360api/v1/charging/{msisdn}/{amount}
//Description: To directly charge the subscriber without sending a consent
//Method: GET
//Headers: Authorization: Bearer {token}
//Required parameters: msisdn, amount
//Response: {"response":{"message":"successfully processed","code":"2006"}}
//Sample Request: {URL}/m360api/v1/charging/9171234567/P1
func (y *Yondu) Charge(msisdn, amount string) (err error) {
	return y.call(y.conf.APIUrl+"/charging/"+msisdn+"/"+amount, 2006)
}

//API URL:  {URL}/m360api/v1/invalid/{msisdn}/{content}
//Description: To send SMS to the subscriber based from your content
//Method: GET
//Headers: Authorization: Bearer {token}
//Required parameters: msisdn
//Sample Request: {URL}/m360api/v1/invalid/9171234567/Hello world!
func (y *Yondu) MT(msisdn, text string) (err error) {
	return y.call(y.conf.APIUrl+"/invalid/"+msisdn+"/"+text, 2006)
}

func (y *Yondu) call(url string, code int) (err error) {
	y.client = &http.Client{
		Timeout: time.Duration(y.conf.Timeout) * time.Second,
	}
	var req *http.Request
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		err = fmt.Errorf("http.NewRequest: %s", err.Error())
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}
	req.SetBasicAuth(y.conf.Auth.User, y.conf.Auth.Pass)

	var resp *http.Response
	resp, err = y.client.Do(req)
	if err != nil {
		err := fmt.Errorf("client.Do: %s", err.Error())
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}

	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err := fmt.Errorf("ioutil.ReadAll: %s", err.Error())
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}

	var yonduResponse SendConsentResponse
	if err = json.Unmarshal(bodyText, yonduResponse); err != nil {
		err := fmt.Errorf("ioutil.ReadAll: %s", err.Error())
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}

	if yonduResponse.Response.Code == code {
		log.WithFields(log.Fields{
			"response": y.conf.ResponseCode[yonduResponse.Response.Code],
			"code":     yonduResponse.Response.Code,
		}).Debug("success")
		return nil
	}

	codeStatus, ok := y.conf.ResponseCode[yonduResponse.Response.Code]
	if !ok {
		err := fmt.Errorf("unexpected response code: %d", yonduResponse.Response.Code)

		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}
	log.WithFields(log.Fields{
		"status": codeStatus,
		"code":   yonduResponse.Response.Code,
	}).Debug("received code")

	return fmt.Errorf("%s: %d", codeStatus, yonduResponse.Response.Code)
}
