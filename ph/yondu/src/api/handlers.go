package yondo

import (
	"encoding/json"
	"fmt"
	//
	//log "github.com/Sirupsen/logrus"
	//smpp_client "github.com/fiorix/go-smpp/smpp"
	//"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
	//"github.com/gin-gonic/gin"

	m "github.com/vostrok/operator/ph/yondu/src/metrics"
	transaction_log_service "github.com/vostrok/qlistener/src/service"
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
