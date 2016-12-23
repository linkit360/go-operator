package metrics

import (
	"time"

	//"github.com/prometheus/client_golang/prometheus"

	m "github.com/vostrok/utils/metrics"
)

var (
	Success        m.Gauge
	Errors         m.Gauge
	WrongParameter m.Gauge
	Dropped        m.Gauge
	Empty          m.Gauge

	ChargeRequestSuccess m.Gauge
	ChargeRequestErrors  m.Gauge

	MTRequestSuccess m.Gauge
	MTRequestErrors  m.Gauge

	SentConsentSuccess m.Gauge
	SentConsentErrors  m.Gauge

	MOSuccess m.Gauge
	MOErrors  m.Gauge

	CallbackSuccess m.Gauge
	CallbackErrors  m.Gauge
)

func Init(appName string) {

	Success = m.NewGauge("", "", "success", "success")
	Errors = m.NewGauge("", "", "errors", "errors")
	Dropped = m.NewGauge("", appName, "dropped", "yondu queue dropped")
	Empty = m.NewGauge("", appName, "empty", "yondu queue empty")

	ChargeRequestSuccess = m.NewGauge("", appName, "charge_success", "yondu charge req")
	ChargeRequestErrors = m.NewGauge("", appName, "charge_errors", "yondu charge req errors")

	MTRequestSuccess = m.NewGauge("", appName, "mt_success", "yondu mt req success")
	MTRequestErrors = m.NewGauge("", appName, "mt_errors", "yondu api out errors")

	SentConsentSuccess = m.NewGauge("", appName, "sentconsent_success", "yondu api out errors")
	SentConsentErrors = m.NewGauge("", appName, "sentconsent_errors", "yondu api out errors")

	MOSuccess = m.NewGauge("", appName, "mo_success", "yondu mo success")
	MOErrors = m.NewGauge("", appName, "mo_errors", "yondu mo errors")

	CallbackSuccess = m.NewGauge("", appName, "callback_success", "yondu callback success")
	CallbackErrors = m.NewGauge("", appName, "callback_errors", "yondu callback errors")

	go func() {
		for range time.Tick(time.Minute) {
			Success.Update()
			Errors.Update()
			WrongParameter.Update()
			Dropped.Update()
			Empty.Update()

			ChargeRequestSuccess.Update()
			ChargeRequestErrors.Update()

			MTRequestSuccess.Update()
			MTRequestErrors.Update()

			SentConsentSuccess.Update()
			SentConsentErrors.Update()

			MOSuccess.Update()
			MOErrors.Update()

			CallbackSuccess.Update()
			CallbackErrors.Update()
		}
	}()
}
