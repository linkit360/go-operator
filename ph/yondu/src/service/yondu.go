package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	"github.com/vostrok/operator/ph/yondu/src/config"
	m "github.com/vostrok/operator/ph/yondu/src/metrics"
	logger "github.com/vostrok/utils/log"
)

type Yondu struct {
	conf        config.YonduConfig
	ThrottleMT  <-chan time.Time
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
}

func initYondu(yConf config.YonduConfig) *Yondu {
	y := &Yondu{
		conf:        yConf,
		responseLog: logger.GetFileLogger(yConf.TransactionLog.ResponseLogPath),
		requestLog:  logger.GetFileLogger(yConf.TransactionLog.RequestLogPath),
	}
	return y
}
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

type YonduResponse struct {
	Response struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"response"`
	ResponseTime time.Time `json:"response_time"`
}

//API URL: {URL}/m360api/v1/consent/{msisdn}/{amount}
//Description: To send transaction code as consent before charging the subscriber
//Method: GET
//Headers: Authorization: Bearer {token}
//Required parameters: msisdn, amount
//Sample Response: {"response":{"message":"verification sent","code":"2001"}}
//Sample Request: {URL}/m360api/v1/consent/9171234567/P1
func (y *Yondu) SendConsent(msisdn, amount string) (YonduResponse, error) {
	return y.call(y.conf.APIUrl+"/consent/"+msisdn+"/"+amount, 2001)
}

//API URL: {URL}/m360api/v1/verify/{msisdn}/{code}
//Description: To verify transaction code sent to subscriber if input is not through SMS
//Method: GET
//Headers: Authorization: Bearer {token}
//Required parameters: msisdn, code
//Sample Response: {"response":{"message":"verification successful","code":"2003"}}
// Sample Request: {URL}/m360api/v1/verify/9171234567/123456
func (y *Yondu) VerifyTransCode(msisdn string, code int) (YonduResponse, error) {
	return y.call(y.conf.APIUrl+"/verify/"+msisdn+"/"+strconv.Itoa(code), 2003)
}

//API URL: {URL}/m360api/v1/charging/{msisdn}/{amount}
//Description: To directly charge the subscriber without sending a consent
//Method: GET
//Headers: Authorization: Bearer {token}
//Required parameters: msisdn, amount
//Response: {"response":{"message":"successfully processed","code":"2006"}}
//Sample Request: {URL}/m360api/v1/charging/9171234567/P1
func (y *Yondu) Charge(msisdn, amount string) (YonduResponse, error) {
	return y.call(y.conf.APIUrl+"/charging/"+msisdn+"/"+amount, 2006)
}

//API URL:  {URL}/m360api/v1/invalid/{msisdn}/{content}
//Description: To send SMS to the subscriber based from your content
//Method: GET
//Headers: Authorization: Bearer {token}
//Required parameters: msisdn
//Sample Request: {URL}/m360api/v1/invalid/9171234567/Hello world!
func (y *Yondu) MT(msisdn, text string) (YonduResponse, error) {
	return y.call(y.conf.APIUrl+"/invalid/"+msisdn+"/"+text, 2006)
}

func (y *Yondu) call(url string, code int) (yonduResponse YonduResponse, err error) {
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
		err = fmt.Errorf("client.Do: %s", err.Error())
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}

	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("ioutil.ReadAll: %s", err.Error())
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}

	if err = json.Unmarshal(bodyText, &yonduResponse); err != nil {
		err = fmt.Errorf("ioutil.ReadAll: %s", err.Error())
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
		return
	}

	codeStatus, ok := y.conf.ResponseCode[yonduResponse.Response.Code]
	if !ok {
		err = fmt.Errorf("unexpected response code: %d", yonduResponse.Response.Code)

		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}
	log.WithFields(log.Fields{
		"status": codeStatus,
		"code":   yonduResponse.Response.Code,
	}).Debug("received code")
	return yonduResponse, fmt.Errorf("%s: %d", codeStatus, yonduResponse.Response.Code)
}

//Callback
//API URL: {YourURL}/{msisdn}/{transid}/{timestamp}/{status_code}
//Description: Where we will send charging status of each transaction
//Method: GET
//Required Parameters: msisdn, transid, timestamp, status_code
//Sample Request:
//{YourURL}/?msisdn=9171234567&transid=123456&timestamp=20160628024446&status_code=1

type CallbackParameters struct {
	Msisdn     string `json:"msisdn"`
	TransID    string `json:"transid"`
	Timestamp  string `json:"timestamp"`
	StatusCode string `json:"status_code"`
}

func (y *Yondu) Callback(c *gin.Context) {
	p := CallbackParameters{}
	var ok bool
	p.Msisdn, ok = c.GetQuery("msisdn")
	if !ok {
		absentParameter("msisdn", c)
		return
	}
	p.TransID, ok = c.GetQuery("transid")
	if !ok {
		absentParameter("transid", c)
		return
	}
	p.Timestamp, ok = c.GetQuery("timestamp")
	if !ok {
		absentParameter("timestamp", c)
		return
	}
	p.StatusCode, ok = c.GetQuery("status_code")
	if !ok {
		absentParameter("status_code", c)
		return
	}
	if err := svc.publishCallback(p); err != nil {
		log.WithFields(log.Fields{
			"p":     fmt.Sprintf("%#v", p),
			"error": err.Error(),
		}).Error("sent callback failed")
	} else {
		log.WithFields(log.Fields{
			"msisdn":  p.Msisdn,
			"transId": p.TransID,
		}).Info("sent")
	}
}

//MO
//API URL: {YourURL}/{msisdn}/{message}/{transid}/{timestamp}
//Description: Where we will send actual message sent by subscriber to 2910
//Method: GET
//Required Parameters: msisdn, message, transid, timestamp
//Sample Request:
//{YourURL}/?msisdn=9171234567&message=Yourkeyword5 26633&transid=123456&timestamp=20160628024446
type MOParameters struct {
	Msisdn    string `json:"msisdn"`
	TransID   string `json:"transid"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

func (y *Yondu) MO(c *gin.Context) {
	p := MOParameters{}
	var ok bool
	p.Msisdn, ok = c.GetQuery("msisdn")
	if !ok {
		absentParameter("msisdn", c)
		return
	}
	p.TransID, ok = c.GetQuery("transid")
	if !ok {
		absentParameter("transid", c)
		return
	}
	p.Timestamp, ok = c.GetQuery("timestamp")
	if !ok {
		absentParameter("timestamp", c)
		return
	}
	p.Message, ok = c.GetQuery("message")
	if !ok {
		absentParameter("message", c)
		return
	}
	if err := svc.publishMO(p); err != nil {
		log.WithFields(log.Fields{
			"p":     fmt.Sprintf("%#v", p),
			"error": err.Error(),
		}).Error("sent mo failed")
	} else {
		log.WithFields(log.Fields{
			"msisdn":  p.Msisdn,
			"transId": p.TransID,
		}).Info("sent")
	}
}

func absentParameter(name string, c *gin.Context) {
	m.WrongParameter.Inc()

	err := fmt.Errorf("Cannot find: %s", name)
	log.WithFields(log.Fields{
		"error": err.Error(),
	}).Error("wrong param")
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": err.Error(),
	})
}

// just log and count all requests
func AccessHandler(c *gin.Context) {
	begin := time.Now()
	c.Next()

	responseTime := time.Since(begin)
	fields := log.Fields{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"req":    c.Request.URL.RawQuery,
		"since":  responseTime,
	}
	if len(c.Errors) > 0 {
		fields["error"] = c.Errors.String()
		log.WithFields(fields).Error("failed")
	} else {
		log.WithFields(fields).Info("access")
	}
	c.Header("X-Response-Time", responseTime.String())
}
