package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	m "github.com/linkit360/go-utils/metrics"
)

var (
	SinceSuccessPaid     prometheus.Gauge
	SmsSuccess           m.Gauge
	SmsError             m.Gauge
	BalanceCheckErrors   m.Gauge
	BalanceCheckSuccess  m.Gauge
	ChargeSuccess        m.Gauge
	ChargeErrors         m.Gauge
	Errors               m.Gauge
	Dropped              m.Gauge
	Empty                m.Gauge
	BalanceCheckDuration prometheus.Summary
	ChargeDuration       prometheus.Summary
)

func newDuration(name string) prometheus.Summary {
	return m.NewSummary(name+"_duration_seconds", name)
}

func Init(appName string) {

	SinceSuccessPaid = m.PrometheusGauge("", "", "since_success_paid", "mobilink since success paid")
	SmsSuccess = m.NewGauge("", "", "sms_success", "sms check success")
	SmsError = m.NewGauge("", "", "sms_errors", "sms check errors")
	BalanceCheckSuccess = m.NewGauge("", "", "balance_check_success", "balance check success")
	BalanceCheckErrors = m.NewGauge("", "", "balance_check_errors", "balance check failed")
	ChargeErrors = m.NewGauge("", "", "charge_errors", "charge has failed")
	ChargeSuccess = m.NewGauge("", "", "charge_success", "charge ok")
	Errors = m.NewGauge("", "", "errors", "errors")
	Dropped = m.NewGauge("", "", "dropped", "mobilink queue dropped")
	Empty = m.NewGauge("", "", "empty", "mobilink queue empty")
	BalanceCheckDuration = newDuration("balance_check")
	ChargeDuration = newDuration("charge_check")

	go func() {
		for range time.Tick(time.Second) {
			SinceSuccessPaid.Inc()
		}
	}()

	go func() {
		for range time.Tick(time.Minute) {
			SmsSuccess.Update()
			SmsError.Update()
			BalanceCheckSuccess.Update()
			BalanceCheckErrors.Update()
			ChargeErrors.Update()
			ChargeSuccess.Update()
			Errors.Update()
			Dropped.Update()
			Empty.Update()
		}
	}()
}
