package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	m "github.com/linkit360/go-utils/metrics"
)

var (
	Success      m.Gauge
	Errors       m.Gauge
	PageNotFound m.Gauge
	NotifyErrors m.Gauge
	MTErrors     prometheus.Gauge
	MO           *MOMetrics
	DN           *DNMetrics
)

type MOMetrics struct {
	AisSuccess      m.Gauge
	DtacSuccess     m.Gauge
	TruehSuccess    m.Gauge
	Unsibscribe     m.Gauge
	UnAuthorized    m.Gauge
	UnknownOperator m.Gauge
	WrongServiceKey m.Gauge
	AbsentParameter m.Gauge
}

type DNMetrics struct {
	AisSuccess                     m.Gauge
	DtacSuccess                    m.Gauge
	TruehSuccess                   m.Gauge
	UnknownOperator                m.Gauge
	WrongServiceKey                m.Gauge
	AbsentParameter                m.Gauge
	MTSuccessfull200               m.Gauge
	MTSentToQueueSuccessfully100   m.Gauge
	MTRejected500                  m.Gauge
	MessageFormatError501          m.Gauge
	UnknownSubscriber510           m.Gauge
	SubscriberBarred511            m.Gauge
	SubscriberError512             m.Gauge
	OperatorFailure520             m.Gauge
	OperatorCongestion521          m.Gauge
	ChargingError530               m.Gauge
	SubscriberNotEnoughBalance531  m.Gauge
	SubscriberExceededFrequency532 m.Gauge
	OtherError550                  m.Gauge
	UnknownCode                    m.Gauge
}

func Init(appName string) {

	Success = m.NewGauge("", "", "success", "success")
	Errors = m.NewGauge("", "", "errors", "errors")
	PageNotFound = m.NewGauge("", appName, "404_page_not_found", "404 page not found")
	NotifyErrors = m.NewGauge("", appName, "notify_errors", "notify errors")
	MTErrors = m.PrometheusGauge("", appName, "mt_errors", "mt errors")

	MO = &MOMetrics{
		AisSuccess:      m.NewGauge(appName, "mo", "ais_success", "ais req success"),
		DtacSuccess:     m.NewGauge(appName, "mo", "dtac_success", "dtac success"),
		TruehSuccess:    m.NewGauge(appName, "mo", "trueh_success", "trueh success"),
		Unsibscribe:     m.NewGauge(appName, "mo", "unsubscribe", "unsubscribe"),
		UnAuthorized:    m.NewGauge(appName, "mo", "unauthorized", "unauthorized"),
		UnknownOperator: m.NewGauge(appName, "mo", "operator_unknown", "unknown operator"),
		WrongServiceKey: m.NewGauge(appName, "mo", "wrong_service_key", "mo wrong service key"),
		AbsentParameter: m.NewGauge(appName, "mo", "adsent_parameter", "wrong parameters"),
	}

	DN = &DNMetrics{
		AisSuccess:                     m.NewGauge(appName, "dn", "ais_success", "ais req success"),
		DtacSuccess:                    m.NewGauge(appName, "dn", "dtac_success", "dtac success"),
		TruehSuccess:                   m.NewGauge(appName, "dn", "trueh_success", "trueh success"),
		UnknownOperator:                m.NewGauge(appName, "dn", "operator_unknown", "dn unknown operator"),
		WrongServiceKey:                m.NewGauge(appName, "dn", "wrong_service_key", "dn wrong service key"),
		AbsentParameter:                m.NewGauge(appName, "dn", "adsent_parameter", "dn wrong parameters"),
		MTSuccessfull200:               m.NewGauge(appName, "dn", "mt_successful", "dn mt_successful"),
		MTSentToQueueSuccessfully100:   m.NewGauge(appName, "dn", "mt_sent_to_queue_successfully", "mt sent to queue successfully"),
		MTRejected500:                  m.NewGauge(appName, "dn", "mt_rejected", "dn mt rejected"),
		MessageFormatError501:          m.NewGauge(appName, "dn", "mt_message_format_error", "dn message format error"),
		UnknownSubscriber510:           m.NewGauge(appName, "dn", "subscriber_unknown", "dn subscriber unknown"),
		SubscriberBarred511:            m.NewGauge(appName, "dn", "subscriber_barred", "dn subscriber barred"),
		SubscriberError512:             m.NewGauge(appName, "dn", "subscriber_error", "dn subscriber error"),
		OperatorFailure520:             m.NewGauge(appName, "dn", "operator_failure", "dn operator failure"),
		OperatorCongestion521:          m.NewGauge(appName, "dn", "operator_congestion", "dn operator cognession"),
		ChargingError530:               m.NewGauge(appName, "dn", "charge_error", "dn charge error"),
		SubscriberNotEnoughBalance531:  m.NewGauge(appName, "dn", "subscriber_not_enough_balance", "dn subscriber not enough balance"),
		SubscriberExceededFrequency532: m.NewGauge(appName, "dn", "subscriber_exceeded_frequency", "dn subscriber exceeded frequency"),
		OtherError550:                  m.NewGauge(appName, "dn", "other_error", "dn other error"),
		UnknownCode:                    m.NewGauge(appName, "dn", "unknown_code", "dn unknown code"),
	}
	go func() {
		for range time.Tick(time.Minute) {
			Success.Update()
			Errors.Update()
			PageNotFound.Update()
			NotifyErrors.Update()

			MO.AisSuccess.Update()
			MO.DtacSuccess.Update()
			MO.TruehSuccess.Update()
			MO.Unsibscribe.Update()
			MO.UnAuthorized.Update()
			MO.UnknownOperator.Update()
			MO.WrongServiceKey.Update()
			MO.AbsentParameter.Update()

			DN.AisSuccess.Update()
			DN.DtacSuccess.Update()
			DN.TruehSuccess.Update()
			DN.UnknownOperator.Update()
			DN.WrongServiceKey.Update()
			DN.AbsentParameter.Update()
			DN.MTSuccessfull200.Update()
			DN.MTSentToQueueSuccessfully100.Update()
			DN.MTRejected500.Update()
			DN.MessageFormatError501.Update()
			DN.UnknownSubscriber510.Update()
			DN.SubscriberBarred511.Update()
			DN.SubscriberError512.Update()
			DN.OperatorFailure520.Update()
			DN.OperatorCongestion521.Update()
			DN.ChargingError530.Update()
			DN.SubscriberNotEnoughBalance531.Update()
			DN.SubscriberExceededFrequency532.Update()
			DN.OtherError550.Update()
			DN.UnknownCode.Update()
		}
	}()
}
