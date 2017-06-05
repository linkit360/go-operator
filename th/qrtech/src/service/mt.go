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

	mid_client "github.com/linkit360/go-mid/rpcclient"
	m "github.com/linkit360/go-operator/th/qrtech/src/metrics"
	transaction_log_service "github.com/linkit360/go-qlistener/src/service"
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
		if err := svc.internals.Load(svc.conf.QRTech.MtPath); err != nil {
			// nothing, continue
		}
		log.WithFields(log.Fields{
			"len":          len(svc.internals.MTLastAt),
			"now_loc_hour": int(time.Now().In(svc.API.location).Hour()),
		}).Debug("loaded internals")

		services, err := mid_client.GetAllServices()
		if err != nil {
			m.Errors.Inc()
			err = fmt.Errorf("mid_client.GetAllServices: %s", err.Error())
			log.WithFields(log.Fields{
				"err": err.Error(),
			}).Fatal("cannot get all services")
			return
		}

		errorFlag := false
		for _, serviceIns := range services {

			lastAt, ok := svc.internals.MTLastAt[serviceIns.Code]
			if ok {
				if time.Since(lastAt.In(svc.API.location)).Hours() > 23 {
					log.WithFields(log.Fields{
						"service":      serviceIns.Code,
						"last":         lastAt.String(),
						"hours_passed": int(time.Since(lastAt).Hours()),
						"now_loc":      time.Now().In(svc.API.location).String(),
					}).Debug("it's time to call")
				}

				if time.Now().In(svc.API.location).Hour() == svc.conf.QRTech.MTSendAtHour &&
					time.Since(lastAt.In(svc.API.location)).Hours() > 1 {
					// call
					log.WithFields(log.Fields{
						"service":      serviceIns.Code,
						"last":         lastAt.String(),
						"hours_passed": int(time.Since(lastAt).Hours()),
						"now_loc_hour": time.Now().In(svc.API.location).String(),
					}).Debug("it's time to call")
				} else {
					log.WithFields(log.Fields{
						"service":      serviceIns.Code,
						"last_at":      lastAt.String(),
						"since_hours":  time.Since(lastAt.In(svc.API.location)).Hours(),
						"send_at_hour": svc.conf.QRTech.MTSendAtHour,
						"now_loc_hour": time.Now().In(svc.API.location).Hour(),
						"hours_passed": int(time.Since(lastAt).Hours()),
					}).Debug("skip")
					continue
				}
			} else {
				log.WithFields(log.Fields{
					"service": serviceIns.Code,
				}).Debug("no notice in config, call")
			}

			now := time.Now().In(svc.API.location)
			interval := 60*now.Hour() + now.Minute()
			if serviceIns.PeriodicAllowedFrom < interval && serviceIns.PeriodicAllowedTo >= interval {
				err := qr.mt(serviceIns.Code, serviceIns.SMSOnContent)
				if err != nil {
					m.MTErrors.Set(1)
					errorFlag = true
				} else {
					log.WithFields(log.Fields{
						"service": serviceIns.Code,
					}).Debug("set now")
					if svc.internals.MTLastAt == nil {
						svc.internals.MTLastAt = make(map[string]time.Time)
					}
					svc.internals.MTLastAt[serviceIns.Code] = now
				}
			} else {
				log.WithFields(log.Fields{
					"service":  serviceIns.Code,
					"interval": interval,
					"from":     serviceIns.PeriodicAllowedFrom,
					"to":       serviceIns.PeriodicAllowedTo,
				}).Debug("not in periodic send interval")
			}
		}

		if !errorFlag {
			m.MTErrors.Set(0)
			log.WithFields(log.Fields{
				"time": svc.internals.MTLastAt,
			}).Debug("overall services: save time last sent MT")
			if err := svc.internals.Save(svc.conf.QRTech.MtPath); err != nil {
				continue
			}
			time.Sleep(time.Second)
		}
	}
}

func (qr *QRTech) mt(serviceCode, smsText string) (err error) {
	v := url.Values{}
	var resp *http.Response
	var qrTechResponse []byte

	v.Add("username", qr.conf.MT.UserName)
	v.Add("serviceid", serviceCode)
	v.Add("broadcastdate", time.Now().Format("20060102150405")[:8])
	v.Add("ctype", "1") // 1=text/ 2=unicode type
	time2Add2Text := time.Now().Format("02-01-2006")
	v.Add("content", strconv.QuoteToASCII(time2Add2Text+" "+smsText)) // when sees any non-ascii type, converts to unicode
	//v.Add("header", 1) // is not mandatory

	log.WithFields(log.Fields{
		"url":    qr.conf.MT.APIUrl,
		"params": v.Encode(),
	}).Debug("call...")

	urlReq := qr.conf.MT.APIUrl + "?" + v.Encode()
	req, err := http.NewRequest("POST", urlReq, strings.NewReader(v.Encode()))
	if err != nil {
		err = fmt.Errorf("Cann't create request: %s", err.Error())
		return
	}
	req.Header.Add("User-Agent", "linkit")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.PostForm = v
	req.Close = false

	// write(9, "POST /QRPartner_API/linkit360/insertContent.php HTTP/1.1\r\nHost: funspaz.com\r\nUser-Agent: Go-http-client/1.1\r\nContent-Length: 154\r\nAccept-Encoding: gzip\r\n\r\nbroadcastdate=20170221&content=%22You+can+got+it+here%3A+http%3A%2F%2Fplatform.th.linkit360.ru%2Fu%2Fget%22&ctype=2&serviceid=421924601&username=LinkIT360", 309) = 309
	// read(9, "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nServer: Microsoft-IIS/7.5\r\nX-Powered-By: ASP.NET\r\nX-Powered-By-Plesk: PleskWin\r\nDate: Tue, 21 Feb 2017 09:31:41 GMT\r\nConnection: close\r\nContent-Length: 21\r\n\r\nInvalid Request Form!", 4096) = 221
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
	operatorToken := ""
	if _, err = strconv.Atoi(result); err != nil {
		log.WithFields(log.Fields{
			"response": result,
			"req":      urlReq,
		}).Error("unknown response")
		return
	} else {
		var negativeAnswer bool
		operatorToken, negativeAnswer = svc.API.conf.MT.ResultCode[result]
		if !negativeAnswer {
			log.WithFields(log.Fields{
				"token":    operatorToken,
				"response": result,
				"req":      urlReq,
			}).Debug("ok")
		} else {
			err = fmt.Errorf("Operator Error Code: %s", operatorToken)
			log.WithFields(log.Fields{
				"response": result,
				"req":      urlReq,
			}).Error(operatorToken)
			return
		}
	}

	operatorErr := ""
	if err != nil {
		operatorErr = err.Error()
	}
	tl := transaction_log_service.OperatorTransactionLog{
		Tid:              operatorToken,
		OperatorToken:    operatorToken,
		Error:            operatorErr,
		ServiceCode:      serviceCode,
		CampaignCode:     serviceCode,
		RequestBody:      qr.conf.MT.APIUrl + "?" + v.Encode(),
		ResponseBody:     string(qrTechResponse),
		ResponseDecision: operatorToken,
		ResponseCode:     resp.StatusCode,
		SentAt:           time.Now().UTC(),
		Notice:           "",
		Type:             "mt",
	}
	if err = svc.publishTransactionLog(tl); err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("sent to transaction log failed")
	}

	return
}
