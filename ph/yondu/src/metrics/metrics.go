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
	APIOutErrors   m.Gauge
	APIOutSuccess  m.Gauge
	APIInErrors    m.Gauge
	APIINSuccess   m.Gauge
	Dropped        m.Gauge
	Empty          m.Gauge
)

func Init(appName string) {
	m.Init(appName)

	Success = m.NewGauge("", "", "success", "success")
	Errors = m.NewGauge("", "", "errors", "errors")
	Dropped = m.NewGauge("", "", "dropped", "yondu queue dropped")
	Empty = m.NewGauge("", "", "empty", "yondu queue empty")
	APIOutErrors = m.NewGauge("", "api_out", "errors", "yondu api out errors")
	APIOutSuccess = m.NewGauge("", "api_out", "", "yondu api out success")
	APIInErrors = m.NewGauge("", "api_in", "errors", "yondu api out errors")
	APIINSuccess = m.NewGauge("", "api_in", "", "yondu api out success")

	go func() {
		for range time.Tick(time.Minute) {
			Success.Update()
			Errors.Update()
			WrongParameter.Update()
			Dropped.Update()
			Empty.Update()
			APIOutErrors.Update()
			APIOutSuccess.Update()
			APIInErrors.Update()
			APIINSuccess.Update()

		}
	}()
}
