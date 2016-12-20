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

	Success = m.NewGauge("", "", "success", "success")
	Errors = m.NewGauge("", "", "errors", "errors")
	Dropped = m.NewGauge("", appName, "dropped", "yondu queue dropped")
	Empty = m.NewGauge("", appName, "empty", "yondu queue empty")
	APIOutErrors = m.NewGauge("", appName, "api_out_errors", "yondu api out errors")
	APIOutSuccess = m.NewGauge("", appName, "api_out_success", "yondu api out success")
	APIInErrors = m.NewGauge("", appName, "api_in_errors", "yondu api in errors")
	APIINSuccess = m.NewGauge("", appName, "api_in_success", "yondu api in success")

	go func() {
		for range time.Tick(time.Minute) {
			//Success.Update()
			//Errors.Update()
			//WrongParameter.Update()
			//Dropped.Update()
			//Empty.Update()
			//APIOutErrors.Update()
			//APIOutSuccess.Update()
			//APIInErrors.Update()
			//APIINSuccess.Update()
		}
	}()
}
