package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	m "github.com/vostrok/utils/metrics"
)

var (
	Success          m.Gauge
	Errors           m.Gauge
	AbsentParameter  m.Gauge
	PageNotFound     m.Gauge
	Dropped          m.Gauge
	Empty            m.Gauge
	NotifyErrors     m.Gauge
	MOParseTimeError m.Gauge
	WrongServiceKey  m.Gauge
	UnknownOperator  m.Gauge
	UnAuthorized     m.Gauge
	Unsibscribe      m.Gauge

	AisSuccess m.Gauge
	AisErrors  m.Gauge

	DtacSuccess m.Gauge
	DtacErrors  m.Gauge

	TruehSuccess m.Gauge
	TruehErrors  m.Gauge

	MTErrors prometheus.Gauge

	DN *DNMetrics
)

type DNMetrics struct {
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
	AbsentParameter = m.NewGauge("", appName, "adsent_parameter", "wrong parameters")
	PageNotFound = m.NewGauge("", appName, "404_page_not_found", "404 page not found")
	Dropped = m.NewGauge("", appName, "dropped", "queue dropped")
	Empty = m.NewGauge("", appName, "empty", "queue empty")
	NotifyErrors = m.NewGauge("", appName, "notify_errors", "notify errors")
	MOParseTimeError = m.NewGauge("", appName, "mo_parse_time_errors", "parse time errors")
	WrongServiceKey = m.NewGauge("", appName, "mo_wrong_service_key", "mo wrong service key")
	UnknownOperator = m.NewGauge("", appName, "operator_unknown", "unknown operator")
	UnAuthorized = m.NewGauge("", appName, "unauthorized", "unauthorized")
	Unsibscribe = m.NewGauge("", appName, "unsubscribe", "unsubscribe")

	AisSuccess = m.NewGauge("", appName, "ais_success", "ais req success")
	AisErrors = m.NewGauge("", appName, "ais_errors", "ais req errors")

	DtacSuccess = m.NewGauge("", appName, "dtac_success", "dtac success")
	DtacErrors = m.NewGauge("", appName, "dtac_errors", "dtac errors")

	TruehSuccess = m.NewGauge("", appName, "trueh_success", "trueh success")
	TruehErrors = m.NewGauge("", appName, "trueh_errors", "trueh errors")

	MTErrors = m.PrometheusGauge("", appName, "mt_errors", "mt errors")
	DN = &DNMetrics{
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
			AbsentParameter.Update()
			PageNotFound.Update()
			Dropped.Update()
			Empty.Update()
			NotifyErrors.Update()
			MOParseTimeError.Update()
			WrongServiceKey.Update()
			UnknownOperator.Update()
			UnAuthorized.Update()
			Unsibscribe.Update()

			AisSuccess.Update()
			AisErrors.Update()

			DtacSuccess.Update()
			DtacErrors.Update()

			TruehSuccess.Update()
			TruehErrors.Update()

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
