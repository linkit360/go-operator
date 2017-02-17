package service

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	m "github.com/vostrok/operator/th/qrtech/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
)

// MT handler
func AddTestMTHandler(r *gin.Engine) {
	r.Group("/qr/mt/failed").POST("", svc.API.testMTFailed)
	r.Group("/qr/mt/ok").POST("", svc.API.testMTOK)
}
func (qr *QRTech) testMTFailed(c *gin.Context) {
	qr.testMt(c)
	c.Writer.WriteString("-11")
}
func (qr *QRTech) testMTOK(c *gin.Context) {
	qr.testMt(c)
	c.Writer.WriteString("11001213")
}
func (qr *QRTech) testMt(c *gin.Context) {
	userName, _ := c.GetPostForm("username")
	serviceid, _ := c.GetPostForm("serviceid")
	broadcastdate, _ := c.GetPostForm("broadcastdate")
	ctype, _ := c.GetPostForm("ctype")
	content, _ := c.GetPostForm("content")
	f := log.Fields{
		"userName":      userName,
		"serviceid":     serviceid,
		"broadcastdate": broadcastdate,
		"ctype":         ctype,
		"content":       content,
	}
	log.WithFields(f).Info("access")
}

func (qr *QRTech) sendMT() {
	for range time.Tick(time.Minute) {
		if err := svc.internals.Load(svc.conf.QRTech.InternalsPath); err != nil {
			continue
		}
		log.WithFields(log.Fields{
			"time": fmt.Sprintf("%#v", svc.internals.MTLastAt),
		}).Debug("time last sent MT")

		services, err := inmem_client.GetAllServices()
		if err != nil {
			m.Errors.Inc()
			err = fmt.Errorf("inmem_client.GetAllServices: %s", err.Error())
			log.WithFields(log.Fields{
				"err": err.Error(),
			}).Error("cannot get all services")
			m.MTErrors.Inc()
			return
		}

		for _, serviceIns := range services {
			lastAt, ok := svc.internals.MTLastAt[serviceIns.Id]
			if ok {
				if time.Since(lastAt.In(svc.API.location)).Hours() < 24 {
					log.WithFields(log.Fields{
						"hours": fmt.Sprintf("%#v", time.Since(lastAt).Hours()),
					}).Debug("no")
					continue
				}
			}

			now := time.Now().In(svc.API.location)
			interval := 60*now.Hour() + now.Minute()
			if serviceIns.PeriodicAllowedFrom < interval && serviceIns.PeriodicAllowedTo >= interval {
				err := qr.mt(serviceIns.Id, serviceIns.SendContentTextTemplate)
				if err != nil {
					m.MTErrors.Inc()
				} else {
					svc.internals.MTLastAt[serviceIns.Id] = now
				}
			}
		}
		log.WithFields(log.Fields{
			"time": svc.internals.MTLastAt,
		}).Debug("save time last sent MT")
		if err := svc.internals.Save(svc.conf.QRTech.InternalsPath); err != nil {
			continue
		}
	}
}

func (qr *QRTech) mt(serviceId int64, smsText string) (err error) {
	logCtx := log.WithFields(log.Fields{
		"q": svc.conf.QRTech.Queue.MT,
	})
	v := url.Values{}
	var resp *http.Response
	var qrTechResponse []byte

	v.Add("username", qr.conf.MT.UserName)
	v.Add("serviceid", strconv.FormatInt(serviceId, 10))
	v.Add("broadcastdate", time.Now().Format("20060102150405")[:8])
	v.Add("ctype", "2")                             // unicode type
	v.Add("content", strconv.QuoteToASCII(smsText)) // when sees any non-ascii type, converts to unicode
	//v.Add("header", 1) // is not mandatory

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

	result := string(qrTechResponse)
	logCtx.WithFields(log.Fields{
		"response": result,
	}).Debug("got response")

	operatorToken, negativeAnswer := svc.API.conf.MT.ResultCode[result]
	if !negativeAnswer {
		logCtx.WithFields(log.Fields{}).Debug("ok")
	} else {
		m.MTErrors.Inc()
		logCtx.WithFields(log.Fields{}).Error(operatorToken)
	}

	operatorErr := ""
	if err != nil {
		operatorErr = err.Error()
	}
	tl := transaction_log_service.OperatorTransactionLog{
		Tid:              operatorToken,
		OperatorToken:    operatorToken,
		Error:            operatorErr,
		ServiceId:        serviceId,
		CampaignId:       serviceId,
		RequestBody:      qr.conf.MT.APIUrl + "?" + v.Encode(),
		ResponseBody:     string(qrTechResponse),
		ResponseDecision: "",
		ResponseCode:     resp.StatusCode,
		SentAt:           time.Now().UTC(),
		Notice:           "",
		Type:             "mt",
	}
	if err = svc.publishTransactionLog(tl); err != nil {
		logCtx.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("sent to transaction log failed")
	}

	return
}
