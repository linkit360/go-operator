package service

// only sends to yondu, async
import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	m "github.com/vostrok/operator/ph/yondu/src/metrics"
	"github.com/vostrok/utils/rec"
)

type EventNotifyResponse struct {
	EventName string        `json:"event_name,omitempty"`
	EventData YonduResponse `json:"event_data,omitempty"`
}

func processMO(deliveries <-chan amqp.Delivery) {
	for msg := range deliveries {

		log.WithFields(log.Fields{
			"priority": msg.Priority,
			"body":     string(msg.Body),
		}).Debug("start process")

		var e EventNotifyResponse
		if err := json.Unmarshal(msg.Body, &e); err != nil {
			m.Dropped.Inc()

			log.WithFields(log.Fields{
				"error": err.Error(),
				"msg":   "dropped",
				"mo":    string(msg.Body),
			}).Error("consume from " + svc.conf.Yondu.Queue.MO.Name)
			goto ack
		}

		t, err := getRecordByMO(e.EventData)
		if err != nil {
		nackGet:
			if err := msg.Nack(false, true); err != nil {
				log.WithFields(log.Fields{
					"mo":    msg.Body,
					"error": err.Error(),
				}).Error("cannot nack")
				time.Sleep(time.Second)
				goto nackGet
			}
			continue
		}

		if err := svc.publishTransactionLog("mo", t); err != nil {
			log.WithFields(log.Fields{
				"event": e.EventName,
				"mo":    msg.Body,
				"error": err.Error(),
			}).Error("sent to transaction log failed")
		nack:
			if err := msg.Nack(false, true); err != nil {
				log.WithFields(log.Fields{
					"mo":    msg.Body,
					"error": err.Error(),
				}).Error("cannot nack")
				time.Sleep(time.Second)
				goto nack
			}
			continue
		} else {
			log.WithFields(log.Fields{
				"queue": svc.conf.Yondu.Queue.CallBack.Name,
				"event": e.EventName,
				"tid":   t.Tid,
			}).Info("success (sent to transaction log)")
		}
	ack:
		if err := msg.Ack(false); err != nil {
			log.WithFields(log.Fields{
				"mo":    msg.Body,
				"error": err.Error(),
			}).Error("cannot ack")
			time.Sleep(time.Second)
			goto ack
		}

	}
}

func getRecordByMO(req MOParameters) (rec.Record, error) {
	r := rec.Record{}
	campaign, err := inmem_client.GetCampaignByKeyWord(req.KeyWord)
	if err != nil {
		m.MOCallUnknownCampaign.Inc()

		err = fmt.Errorf("inmem_client.GetCampaignByKeyWord: %s", err.Error())
		log.WithFields(log.Fields{
			"keyword": req.KeyWord,
			"error":   err.Error(),
		}).Error("cannot get campaign by keyword")
		return r, err
	}
	svc, err := inmem_client.GetServiceById(campaign.ServiceId)
	if err != nil {
		m.MOCallUnknownService.Inc()

		err = fmt.Errorf("inmem_client.GetServiceById: %s", err.Error())
		log.WithFields(log.Fields{
			"keyword":    req.KeyWord,
			"serviceId":  campaign.ServiceId,
			"campaignId": campaign.Id,
			"error":      err.Error(),
		}).Error("cannot get service by id")
		return r, err
	}
	publisher := ""
	pixelSetting, err := inmem_client.GetPixelSettingByCampaignId(campaign.Id)
	if err != nil {
		m.MOCallUnknownPublisher.Inc()

		err = fmt.Errorf("inmem_client.GetPixelSettingByCampaignId: %s", err.Error())
		log.WithFields(log.Fields{
			"keyword":    req.KeyWord,
			"serviceId":  campaign.ServiceId,
			"campaignId": campaign.Id,
			"error":      err.Error(),
		}).Error("cannot get pixel setting by campaign id")
	} else {
		publisher = pixelSetting.Publisher
	}
	r = rec.Record{
		Msisdn:             req.Msisdn,
		Tid:                rec.GenerateTID(),
		SubscriptionStatus: "",
		CountryCode:        "ph",
		OperatorCode:       "51000",
		Publisher:          publisher,
		Pixel:              "",
		CampaignId:         campaign.Id,
		ServiceId:          campaign.ServiceId,
		DelayHours:         svc.DelayHours,
		PaidHours:          svc.PaidHours,
		KeepDays:           svc.KeepDays,
		Price:              100 * int(svc.Price),
	}
	return r, nil
}
