package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	m "github.com/vostrok/utils/metrics"
)

var (
	ResponseCodes *ResponseCodesMetrics

	Success         m.Gauge
	Errors          m.Gauge
	AbsentParameter m.Gauge
	PageNotFound    m.Gauge
	Dropped         m.Gauge
	Empty           m.Gauge

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

	MTDuration      prometheus.Summary
	ChargeDuration  prometheus.Summary
	ConsentDuration prometheus.Summary
)

type ResponseCodesMetrics struct {
	VerificationSent2001       m.Gauge
	InvalidMobileNumber2002    m.Gauge
	VerificationSuccessful2003 m.Gauge
	InvalidCodeCombination2004 m.Gauge
	ChargedFailed2005          m.Gauge
	SuccessfullyProcessed2006  m.Gauge
	InvalidTariff2007          m.Gauge
	UnknownCode                m.Gauge
}

func Init(appName string) {
	ResponseCodes = &ResponseCodesMetrics{
		VerificationSent2001:       m.NewGauge("", appName, "verification_sent", "response code verification sent"),
		InvalidMobileNumber2002:    m.NewGauge("", appName, "invalid_mobile_number", "response code invalid mobile number"),
		VerificationSuccessful2003: m.NewGauge("", appName, "verification_successful", "response code verification successful"),
		InvalidCodeCombination2004: m.NewGauge("", appName, "invalid_code_combination", "response code invalid code combination"),
		ChargedFailed2005:          m.NewGauge("", appName, "charged_failed", "response code charged failed"),
		SuccessfullyProcessed2006:  m.NewGauge("", appName, "successfully_processed", "response code successfully processed"),
		InvalidTariff2007:          m.NewGauge("", appName, "invalid_tariff", "response code invalid tariff"),
		UnknownCode:                m.NewGauge("", appName, "unknown_code", "response code unknown"),
	}

	Success = m.NewGauge("", "", "success", "success")
	Errors = m.NewGauge("", "", "errors", "errors")
	AbsentParameter = m.NewGauge("", appName, "adsent_parameter", "wrong parameters")
	PageNotFound = m.NewGauge("", appName, "404_page_not_found", "404 page not found")
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

	MTDuration = m.NewSummary(appName+"_mt_duration_seconds", "mt duration seconds")
	ChargeDuration = m.NewSummary(appName+"_charge_duration_seconds", "charge duration seconds")
	ConsentDuration = m.NewSummary(appName+"_consent_duration_seconds", "consent duration seconds")

	go func() {
		for range time.Tick(time.Minute) {
			ResponseCodes.VerificationSent2001.Update()
			ResponseCodes.InvalidMobileNumber2002.Update()
			ResponseCodes.VerificationSuccessful2003.Update()
			ResponseCodes.InvalidCodeCombination2004.Update()
			ResponseCodes.ChargedFailed2005.Update()
			ResponseCodes.SuccessfullyProcessed2006.Update()
			ResponseCodes.InvalidTariff2007.Update()
			ResponseCodes.UnknownCode.Update()

			Success.Update()
			Errors.Update()
			AbsentParameter.Update()
			PageNotFound.Update()
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
