package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	"github.com/linkit360/go-operator/ph/yondu/src/config"
	m "github.com/linkit360/go-operator/ph/yondu/src/metrics"
	logger "github.com/linkit360/go-utils/log"
	"github.com/linkit360/go-utils/rec"
)

// Yondu API implements:
// DN, MO, XPortal Partner API requests

type Yondu struct {
	conf        config.YonduConfig
	Throttle    ThrottleConfig
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
}

type ThrottleConfig struct {
	MT <-chan time.Time
}

func AddHandlers(r *gin.Engine) {
	tgYonduAPI := r.Group("/api/")
	tgYonduAPI.Group("/mo").GET("", AccessHandler, svc.YonduAPI.MO)
	tgYonduAPI.Group("/dn").GET("", AccessHandler, svc.YonduAPI.DN)
}
func AddTestHandlers(r *gin.Engine) {
	tgYonduAPI := r.Group("/yondu/")
	tgYonduAPI.Group("/mt").GET("", AccessHandler, svc.YonduAPI.testAPIcallMT)
}
func (y *Yondu) testAPIcallMT(c *gin.Context) {
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte(`{"status_code":200,"msg":"Success."}`))
}
func initYondu(yConf config.YonduConfig) *Yondu {
	y := &Yondu{
		conf: yConf,
		Throttle: ThrottleConfig{
			MT: time.Tick(time.Second / time.Duration(yConf.Throttle.MT+1)),
		},
		responseLog: logger.GetFileLogger(yConf.TransactionLogFilePath.ResponseLogPath),
		requestLog:  logger.GetFileLogger(yConf.TransactionLogFilePath.RequestLogPath),
	}
	return y
}

type YonduResponseExtended struct {
	RequestUrl      string        `json:"request"`
	ResponseCode    int           `json:"response_code"`
	ResponseMessage string        `json:"response_msg"`
	ResponseError   string        `json:"response_error"`
	ResponseRawBody string        `json:"response_raw_body"`
	ResponseTime    time.Time     `json:"response_time"`
	Yondu           YonduResponse `json:"response"`
}

type YonduResponse struct {
	Message string `json:"msg,omitempty"`
	Code    int    `json:"status_code,omitempty"`
}

// sends charge request and if the msisdn successfully charged, then sends content from the message
//{URL}?key={key}&msisdn={msisdn}&keyword={keyword}&message={message}&rrn={rrn }
// /api/dn?telco=globe&msisdn=9951502420&rrn=1488218310&status=SUCCESS&code=201&timestamp=1488189660
func (y *Yondu) MT(r rec.Record) (yR YonduResponseExtended, err error) {
	begin := time.Now()
	logCtx := log.WithFields(log.Fields{
		"tid": r.Tid,
	})

	v := url.Values{}
	v.Add("key", y.conf.AuthKey)
	v.Add("msisdn", r.Msisdn)
	v.Add("rrn", r.OperatorToken)
	apiUrl := ""
	if r.SMSText != "" {
		code, ok := y.conf.TariffCode["0"]
		if !ok {
			logCtx.WithFields(log.Fields{
				"error": "cann't find code to send content",
			}).Error("cannot process")
			m.WrongTariff.Inc()
			m.Errors.Inc()
			m.MTRequestErrors.Inc()
			return
		}
		v.Add("keyword", code)
		v.Add("message", r.SMSText)
	} else {
		code, ok := y.conf.TariffCode[strconv.Itoa(r.Price)]
		if !ok {
			logCtx.WithFields(log.Fields{
				"error": "cann't find code for price",
				"price": r.Price,
			}).Error("cannot process")
			m.WrongTariff.Inc()
			m.Errors.Inc()
			m.MTRequestErrors.Inc()
			return
		}
		v.Add("keyword", code)
		v.Add("message", "")

	}
	apiUrl = y.conf.APIUrl + "?" + v.Encode()

	defer func() {
		yR.RequestUrl = apiUrl
		yR.ResponseTime = time.Now().UTC()
		if err != nil {
			yR.ResponseError = err.Error()
		}
	}()
	y.client = &http.Client{
		Timeout: time.Duration(y.conf.Timeout) * time.Second,
	}
	var req *http.Request
	req, err = http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		err = fmt.Errorf("http.NewRequest: %s", err.Error())
		logCtx.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}
	req.Close = false

	var resp *http.Response
	resp, err = y.client.Do(req)
	if err != nil {
		err = fmt.Errorf("client.Do: %s", err.Error())
		logCtx.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}
	yR.ResponseCode = resp.StatusCode
	if resp.StatusCode != 200 {
		err = fmt.Errorf("status code: %d", resp.StatusCode)
		logCtx.WithFields(log.Fields{
			"error":  err.Error(),
			"status": resp.Status,
		}).Error("cannot process")
		return
	}

	bodyText, err := ioutil.ReadAll(resp.Body)
	logCtx.WithFields(log.Fields{
		"body":           string(bodyText),
		"respStatusCode": resp.StatusCode,
		"url":            apiUrl,
	}).Debug("")

	yR.ResponseRawBody = string(bodyText)

	if err != nil {
		err = fmt.Errorf("ioutil.ReadAll: %s", err.Error())
		logCtx.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}

	var responseJson YonduResponse
	if err = json.Unmarshal(bodyText, &responseJson); err != nil {
		err = fmt.Errorf("ioutil.ReadAll: %s", err.Error())
		logCtx.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("cannot process")
		return
	}
	yR.Yondu = responseJson
	codeStatus, ok := y.conf.MTResponseCode[strconv.Itoa(responseJson.Code)]
	if !ok {
		m.MTRequestUnknownCode.Inc()
		yR.ResponseError =
			fmt.Errorf("unexpected response code: %s", responseJson.Code).Error()

		logCtx.WithFields(log.Fields{
			"code":  responseJson.Code,
			"error": "unexpected response code",
		}).Error("cannot process")
		yR.ResponseMessage = "invalid code"
		return
	} else {
		yR.ResponseMessage = codeStatus
	}
	logCtx.WithFields(log.Fields{
		"status": codeStatus,
		"code":   responseJson.Code,
	}).Debug("received code")

	m.MTDuration.Observe(time.Since(begin).Seconds())
	if err == nil {
		m.Success.Inc()
		m.MTRequestSuccess.Inc()
	} else {
		m.Errors.Inc()
		m.MTRequestErrors.Inc()
	}
	return
}

type DNParameters struct {
	Params struct {
		Telco     string `json:"telco"`
		Msisdn    string `json:"msisdn"`
		RRN       string `json:"rrn"`
		Status    string `json:"status"`
		Code      string `json:"code"`
		Timestamp string `json:"timestamp"`
	}
	Raw string `json:"req_url"`
	Tid string `json:"tid"`
}

func (y *Yondu) DN(c *gin.Context) {

	p := DNParameters{
		Raw: c.Request.URL.Path + "/?" + c.Request.URL.RawQuery,
		Tid: rec.GenerateTID(),
	}
	logCtx := log.WithFields(log.Fields{
		"tid": p.Tid,
	})
	c.Set("tid", p.Tid)

	var ok bool
	p.Params.Telco, ok = c.GetQuery("telco")
	if !ok {
		m.Errors.Inc()
		m.DNErrors.Inc()

		absentParameter(p.Tid, "telco", c)
		return
	}
	p.Params.Msisdn, ok = c.GetQuery("msisdn")
	if !ok {
		m.Errors.Inc()
		m.DNErrors.Inc()

		absentParameter(p.Tid, "msisdn", c)
		return
	}
	p.Params.RRN, ok = c.GetQuery("rrn")
	if !ok {
		m.Errors.Inc()
		m.DNErrors.Inc()

		absentParameter(p.Tid, "rrn", c)
		return
	}
	p.Params.Status, ok = c.GetQuery("status")
	if !ok {
		m.Errors.Inc()
		m.DNErrors.Inc()

		absentParameter(p.Tid, "status", c)
		return
	}
	p.Params.Code, ok = c.GetQuery("code")
	if !ok {
		m.Errors.Inc()
		m.DNErrors.Inc()

		absentParameter(p.Tid, "code", c)
		return
	}
	dnMessage, _ := y.conf.DNCode[p.Params.Code]

	p.Params.Timestamp, ok = c.GetQuery("timestamp")
	if !ok {
		m.Errors.Inc()
		m.DNErrors.Inc()

		absentParameter(p.Tid, "timestamp", c)
		return
	}
	logIncoming("dn", p)
	if err := svc.publishDN(p); err != nil {
		m.Errors.Inc()
		m.DNErrors.Inc()

		logCtx.WithFields(log.Fields{
			"p":     fmt.Sprintf("%#v", p),
			"error": err.Error(),
		}).Error("sent dn failed")
	} else {
		m.Success.Inc()
		m.DNSuccess.Inc()

		logCtx.WithFields(log.Fields{
			"msisdn": p.Params.Msisdn,
			"rrn":    p.Params.RRN,
			"dn":     dnMessage,
		}).Info("sent")
	}
}

//MO
//telco=globe&msisdn=9171234567&message=sample%20keyword&rrn=abcd1234
type MOParameters struct {
	Params struct {
		Telco   string `json:"telco"`
		Msisdn  string `json:"msisdn"`
		Message string `json:"message"`
		RRN     string `json:"rrn"`
	}
	Raw        string `json:"req_url"`
	Tid        string `json:"tid"`
	ReceivedAt time.Time
}

func (y *Yondu) MO(c *gin.Context) {
	p := MOParameters{
		Raw:        c.Request.URL.Path + "/" + c.Request.URL.RawQuery,
		Tid:        rec.GenerateTID(),
		ReceivedAt: time.Now().UTC(),
	}
	logCtx := log.WithFields(log.Fields{
		"tid": p.Tid,
	})
	c.Set("tid", p.Tid)

	var ok bool
	p.Params.Msisdn, ok = c.GetQuery("msisdn")
	if !ok {
		m.Errors.Inc()
		m.MOErrors.Inc()

		absentParameter(p.Tid, "msisdn", c)
		return
	}
	p.Params.Telco, ok = c.GetQuery("telco")
	if !ok {
		m.Errors.Inc()
		m.MOErrors.Inc()
	}
	p.Params.Message, ok = c.GetQuery("message")
	if !ok {
		m.Errors.Inc()
		m.MOErrors.Inc()

		absentParameter(p.Tid, "message", c)
		return
	}
	p.Params.RRN, ok = c.GetQuery("rrn")
	if !ok {
		m.Errors.Inc()
		m.MOErrors.Inc()

		absentParameter(p.Tid, "rrn", c)
		return
	}
	logIncoming("mo", p)
	if err := svc.publishMO(p); err != nil {
		m.Errors.Inc()
		m.MOErrors.Inc()

		logCtx.WithFields(log.Fields{
			"p":     fmt.Sprintf("%#v", p),
			"error": err.Error(),
		}).Error("sent mo failed")
	} else {
		m.Success.Inc()
		m.MOSuccess.Inc()

		logCtx.WithFields(log.Fields{
			"msisdn": p.Params.Msisdn,
			"rrn":    p.Params.RRN,
		}).Info("sent")
	}

}

func absentParameter(tid, name string, c *gin.Context) {
	m.AbsentParameter.Inc()

	err := fmt.Errorf("Cannot find: %s", name)
	log.WithFields(log.Fields{
		"tid":   tid,
		"req":   c.Request.URL.Path,
		"query": c.Request.URL.RawQuery,
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

	path := c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		path = path + "?" + c.Request.URL.RawQuery

	}

	fields := log.Fields{
		"method": c.Request.Method,
		"url":    path,
		"since":  responseTime,
	}
	tid, found := c.Get("tid")
	if found {
		fields["tid"] = tid
	}
	if len(c.Errors) > 0 {
		fields["error"] = c.Errors.String()
		log.WithFields(fields).Error("failed")
	} else {
		log.WithFields(fields).Info("access")
	}
	c.Header("X-Response-Time", responseTime.String())
}
