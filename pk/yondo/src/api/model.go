package yondo

import (
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/vostrok/utils/amqp"
	m "github.com/vostrok/utils/metrics"
)

type Yondo struct {
	conf        Config
	ThrottleMT  <-chan time.Time
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
	notifier    *amqp.Notifier
}
type Config struct {
	Enabled        bool                 `default:"true" yaml:"enabled"`
	SaveRetries    bool                 `default:"true" yaml:"save_retries"`
	MTChanCapacity int                  `default:"1000" yaml:"channel_camacity"`
	Location       string               `default:"Asia/Karachi" yaml:"location"`
	TransactionLog TransactionLogConfig `yaml:"log_transaction"`
}
type TransactionLogConfig struct {
	Queue           string `default:"transaction_log" yaml:"transaction_log"`
	ResponseLogPath string `default:"/var/log/response_yondo.log" yaml:"response"`
	RequestLogPath  string `default:"/var/log/request_yondo.log" yaml:"request"`
}

var (
	SinceSuccessPaid prometheus.Gauge
	smsSuccess       m.Gauge
	smsError         m.Gauge
	chargeSuccess    m.Gauge
	chargeErrors     m.Gauge
	Errors           m.Gauge
)

func initMetrics() {
	SinceSuccessPaid = m.PrometheusGauge("", "", "since_success_paid", "yondo since success paid")

	smsSuccess = m.NewGauge("", "", "sms_success", "sms check success")
	smsError = m.NewGauge("", "", "sms_errors", "sms check errors")
	chargeErrors = m.NewGauge("", "", "charge_errors", "charge has failed")
	chargeSuccess = m.NewGauge("", "", "charge_success", "charge ok")
	Errors = m.NewGauge("", "", "errors", "errors")

	go func() {
		for range time.Tick(time.Second) {
			SinceSuccessPaid.Inc()
		}
	}()
	go func() {
		for range time.Tick(time.Minute) {
			smsSuccess.Update()
			smsError.Update()
			chargeErrors.Update()
			chargeSuccess.Update()
			Errors.Update()
		}
	}()
}

func getLogger(path string) *log.Logger {
	handler, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		log.WithFields(log.Fields{
			"path":  path,
			"error": err.Error(),
		}).Fatal("cannot open file")
	}
	return &log.Logger{
		Out:       handler,
		Formatter: new(log.TextFormatter),
		Hooks:     make(log.LevelHooks),
		Level:     log.DebugLevel,
	}
}
