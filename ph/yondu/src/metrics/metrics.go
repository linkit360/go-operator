package metrics

import (
	"time"

	//"github.com/prometheus/client_golang/prometheus"

	m "github.com/vostrok/utils/metrics"
)

var (
	Success m.Gauge
	Errors  m.Gauge
	Dropped m.Gauge
	Empty   m.Gauge
)

func Init() {

	Success = m.NewGauge("", "", "success", "success")
	Errors = m.NewGauge("", "", "errors", "errors")
	Dropped = m.NewGauge("", "", "dropped", "yondu queue dropped")
	Empty = m.NewGauge("", "", "empty", "yondu queue empty")

	go func() {
		for range time.Tick(time.Minute) {
			Success.Update()
			Errors.Update()
			Dropped.Update()
			Empty.Update()
		}
	}()
}
