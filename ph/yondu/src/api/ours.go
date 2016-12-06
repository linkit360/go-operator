package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	m "github.com/vostrok/operator/ph/yondu/src/metrics"
	//transaction_log_service "github.com/vostrok/qlistener/src/service"
	"github.com/vostrok/utils/amqp"
	//rec "github.com/vostrok/utils/rec"
	"bitbucket.org/carprice/auction/bids-api/api/utils/ip"
)

//Callback
//API URL: {YourURL}/{msisdn}/{transid}/{timestamp}/{status_code}
//Description: Where we will send charging status of each transaction
//Method: GET
//Required Parameters: msisdn, transid, timestamp, status_code
//Sample Request:
//{YourURL}/?msisdn=9171234567&transid=123456&timestamp=20160628024446&status_code=1

//MO
//API URL: {YourURL}/{msisdn}/{message}/{transid}/{timestamp}
//Description: Where we will send actual message sent by subscriber to 2910
//Method: GET
//Required Parameters: msisdn, message, transid, timestamp
//Sample Request:
//{YourURL}/?msisdn=9171234567&message=Yourkeyword5 26633&transid=123456&timestamp=20160628024446
type MOParameters struct {
	Msisdn    string
	Message   string
	TransID   int64
	Timestamp string
}

func (y *Yondu) MO(c *gin.Context) {

	p := MOParameters{}
	var ok bool
	p.Msisdn, ok = c.GetQuery("msisdn")
	if !ok {
		m.WrongParameter.Inc()

		err := fmt.Errorf("Cannot find: %s", "msisdn")
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Debug("cannot get param")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	p.Message, ok = c.GetQuery("message")
	if !ok {
		m.WrongParameter.Inc()

		err := fmt.Errorf("Cannot find: %s", "message")
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Debug("cannot get param")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	p.TransID, ok = c.GetQuery("transid")
	if !ok {
		m.WrongParameter.Inc()

		err := fmt.Errorf("Cannot find: %s", "transid")
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Debug("cannot get param")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	p.Timestamp, ok = c.GetQuery("timestamp")
	if !ok {
		m.WrongParameter.Inc()

		err := fmt.Errorf("Cannot find: %s", "timestamp")
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Debug("cannot get param")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

}
