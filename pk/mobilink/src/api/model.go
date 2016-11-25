package mobilink

import (
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	smpp_client "github.com/fiorix/go-smpp/smpp"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/vostrok/utils/amqp"
	m "github.com/vostrok/utils/metrics"
)

type Mobilink struct {
	conf        Config
	rps         int
	ThrottleMT  <-chan time.Time
	location    *time.Location
	client      *http.Client
	smpp        *smpp_client.Transmitter
	responseLog *log.Logger
	requestLog  *log.Logger
	notifier    *amqp.Notifier
}
type Config struct {
	Rps            int                  `default:"10" yaml:"rps"`
	Enabled        bool                 `default:"true" yaml:"enabled"`
	SaveRetries    bool                 `default:"true" yaml:"save_retries"`
	MTChanCapacity int                  `default:"1000" yaml:"channel_camacity"`
	Location       string               `default:"Asia/Karachi" yaml:"location"`
	TransactionLog TransactionLogConfig `yaml:"log_transaction"`
	Connection     ConnnectionConfig    `yaml:"connection"`
}
type TransactionLogConfig struct {
	Queue           string `default:"transaction_log" yaml:"transaction_log"`
	ResponseLogPath string `default:"/var/log/response_mobilink.log" yaml:"response"`
	RequestLogPath  string `default:"/var/log/request_mobilink.log" yaml:"request"`
}
type ConnnectionConfig struct {
	MT   MTConfig   `yaml:"mt" json:"mt"`
	Smpp SmppConfig `yaml:"smpp" json:"smpp"`
}
type SmppConfig struct {
	ShortNumber string `default:"4162" yaml:"short_number" json:"short_number"`
	Addr        string `default:"182.16.255.46:15019" yaml:"endpoint"`
	User        string `default:"SLYEPPLA" yaml:"user"`
	Password    string `default:"SLYPEE_1" yaml:"pass"`
	Timeout     int    `default:"20" yaml:"timeout"`
}
type MTConfig struct {
	Url                  string            `yaml:"url" default:"http://localhost:50306/mobilink_handler" `
	Headers              map[string]string `yaml:"headers"`
	TimeoutSec           int               `default:"20" yaml:"timeout" json:"timeout"`
	TarifficateBody      string            `yaml:"mt_body"`
	PaidBodyContains     []string          `yaml:"paid_body_contains" json:"paid_body_contains"`
	CheckBalanceBody     string            `yaml:"check_balance_body"`
	PostPaidBodyContains []string          `yaml:"postpaid_body_contains" json:"postpaid_body_contains"`
}

var (
	SMPPConnected       prometheus.Gauge
	SinceSuccessPaid    prometheus.Gauge
	smsSuccess          m.Gauge
	smsError            m.Gauge
	balanceCheckErrors  m.Gauge
	balanceCheckSuccess m.Gauge
	chargeSuccess       m.Gauge
	chargeErrors        m.Gauge
	Errors              m.Gauge
)

func initMetrics() {
	SMPPConnected = m.PrometheusGauge("", "", "smpp_connected", "mobilink smppconnected")
	SinceSuccessPaid = m.PrometheusGauge("", "", "since_success_paid", "mobilink since success paid")

	smsSuccess = m.NewGauge("", "", "sms_success", "sms check success")
	smsError = m.NewGauge("", "", "sms_errors", "sms check errors")
	balanceCheckSuccess = m.NewGauge("", "", "balance_check_success", "balance check success")
	balanceCheckErrors = m.NewGauge("", "", "balance_check_errors", "balance check failed")
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
			balanceCheckSuccess.Update()
			balanceCheckErrors.Update()
			chargeErrors.Update()
			chargeSuccess.Update()
			Errors.Update()
		}
	}()
}

// prefix from table
func My(msisdn string) bool {
	return msisdn[:2] == "92"
}

// todo: chan gap cannot be too big bzs of the size

func Init(
	mobilinkRps int,
	mobilinkConf Config,
	notifier *amqp.Notifier,
) *Mobilink {
	initMetrics()
	mb := &Mobilink{
		rps:  mobilinkRps,
		conf: mobilinkConf,
	}
	log.Info("mb metrics init done")

	mb.notifier = notifier
	mb.ThrottleMT = time.Tick(time.Second / time.Duration(mobilinkConf.Rps))
	mb.requestLog = getLogger(mobilinkConf.TransactionLog.RequestLogPath)
	log.Info("request logger init done")

	mb.responseLog = getLogger(mobilinkConf.TransactionLog.ResponseLogPath)
	log.Info("response logger init done")

	var err error
	mb.location, err = time.LoadLocation(mobilinkConf.Location)
	if err != nil {
		log.WithFields(log.Fields{
			"location": mobilinkConf.Location,
			"error":    err,
		}).Fatal("init location")
	}
	log.Info("location init done")

	mb.client = &http.Client{
		Timeout: time.Duration(mobilinkConf.Connection.MT.TimeoutSec) * time.Second,
	}
	log.Info("http client init done")

	mb.smpp = &smpp_client.Transmitter{
		Addr:        mobilinkConf.Connection.Smpp.Addr,
		User:        mobilinkConf.Connection.Smpp.User,
		Passwd:      mobilinkConf.Connection.Smpp.Password,
		RespTimeout: time.Duration(mobilinkConf.Connection.Smpp.Timeout) * time.Second,
		SystemType:  "SMPP",
	}
	log.Info("smpp client init done")

	connStatus := mb.smpp.Bind()
	go func() {
		for c := range connStatus {
			if c.Status().String() != "Connected" {
				SMPPConnected.Set(0)
				log.WithFields(log.Fields{
					"operator": "mobilink",
					"status":   c.Status().String(),
					"error":    "disconnected:" + c.Status().String(),
				}).Error("smpp moblink connect status")
			} else {
				log.WithFields(log.Fields{
					"operator": "mobilink",
					"status":   c.Status().String(),
				}).Info("smpp moblink connect status")
				SMPPConnected.Set(1)
			}
		}
	}()
	return mb
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
