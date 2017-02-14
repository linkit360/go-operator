package metrics

import (
	"time"

	//"github.com/prometheus/client_golang/prometheus"

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

	AisSuccess m.Gauge
	AisErrors  m.Gauge

	DtacSuccess m.Gauge
	DtacErrors  m.Gauge

	TruehSuccess m.Gauge
	TruehErrors  m.Gauge
)

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
	UnknownOperator = m.NewGauge("", appName, "unknown_operator", "unknown operator")
	UnAuthorized = m.NewGauge("", appName, "unauthorized", "unauthorized")

	AisSuccess = m.NewGauge("", appName, "ais_success", "ais req success")
	AisErrors = m.NewGauge("", appName, "ais_errors", "ais req errors")

	DtacSuccess = m.NewGauge("", appName, "dtac_success", "dtac success")
	DtacErrors = m.NewGauge("", appName, "dtac_errors", "dtac errors")

	TruehSuccess = m.NewGauge("", appName, "trueh_success", "trueh success")
	TruehErrors = m.NewGauge("", appName, "trueh_errors", "trueh errors")

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

			AisSuccess.Update()
			AisErrors.Update()

			DtacSuccess.Update()
			DtacErrors.Update()

			TruehSuccess.Update()
			TruehErrors.Update()
		}
	}()
}
