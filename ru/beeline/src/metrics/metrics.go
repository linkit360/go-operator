package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	m "github.com/linkit360/go-utils/metrics"
)

var (
	Incoming           m.Gauge
	Success            m.Gauge
	Errors             m.Gauge
	AbsentParameter    m.Gauge
	PageNotFound       m.Gauge
	SMPPConnected      prometheus.Gauge
	SMPPReconnectCount prometheus.Gauge
)

func Init(appName string) {

	Success = m.NewGauge("", "", "success", "success")
	Incoming = m.NewGauge("", "", "incoming", "incoming")
	Errors = m.NewGauge("", "", "errors", "errors")
	AbsentParameter = m.NewGauge("", appName, "adsent_parameter", "wrong parameters")
	PageNotFound = m.NewGauge("", appName, "404_page_not_found", "404 page not found")
	SMPPConnected = m.PrometheusGauge("", appName, "smpp_connected", "smpp connected")
	SMPPReconnectCount = m.PrometheusGauge("", appName, "smpp_reconnect_count", "smpp reconnect count")

	go func() {
		for range time.Tick(time.Minute) {
			Incoming.Update()
			Success.Update()
			Errors.Update()
			AbsentParameter.Update()
			PageNotFound.Update()
		}
	}()
}
