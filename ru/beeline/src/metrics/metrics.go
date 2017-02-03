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
		}
	}()
}
