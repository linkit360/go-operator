package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	m "github.com/vostrok/utils/metrics"
)

var (
	Success         m.Gauge
	Errors          m.Gauge
	AbsentParameter m.Gauge
	WrongTariff     m.Gauge
	PageNotFound    m.Gauge
	Dropped         m.Gauge
	Empty           m.Gauge

	MTRequestSuccess     m.Gauge
	MTRequestErrors      m.Gauge
	MTRequestUnknownCode m.Gauge

	MOSuccess m.Gauge
	MOErrors  m.Gauge

	DNSuccess m.Gauge
	DNErrors  m.Gauge

	MTDuration prometheus.Summary
)

func Init(appName string) {
	Success = m.NewGauge("", "", "success", "success")
	Errors = m.NewGauge("", "", "errors", "errors")
	AbsentParameter = m.NewGauge("", appName, "adsent_parameter", "wrong parameters")
	WrongTariff = m.NewGauge("", appName, "wrong_tariff", "wrong tariff")
	PageNotFound = m.NewGauge("", appName, "404_page_not_found", "404 page not found")
	Dropped = m.NewGauge("", appName, "dropped", "yondu queue dropped")
	Empty = m.NewGauge("", appName, "empty", "yondu queue empty")

	MTRequestSuccess = m.NewGauge("", appName, "mt_success", "yondu mt req success")
	MTRequestErrors = m.NewGauge("", appName, "mt_errors", "yondu api out errors")
	MTRequestUnknownCode = m.NewGauge("", appName, "mt_unknown_code", "yondu unknown code")

	MOSuccess = m.NewGauge("", appName, "mo_success", "yondu mo success")
	MOErrors = m.NewGauge("", appName, "mo_errors", "yondu mo errors")

	DNSuccess = m.NewGauge("", appName, "callback_success", "yondu callback success")
	DNErrors = m.NewGauge("", appName, "callback_errors", "yondu callback errors")

	MTDuration = m.NewSummary(appName+"_mt_duration_seconds", "mt duration seconds")
	go func() {
		for range time.Tick(time.Minute) {

			Success.Update()
			Errors.Update()
			AbsentParameter.Update()
			WrongTariff.Update()
			PageNotFound.Update()
			Dropped.Update()
			Empty.Update()

			MTRequestSuccess.Update()
			MTRequestErrors.Update()
			MTRequestUnknownCode.Update()

			MOSuccess.Update()
			MOErrors.Update()

			DNSuccess.Update()
			DNErrors.Update()
		}
	}()
}
