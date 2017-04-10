package mobilink

import (
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/linkit360/go-utils/amqp"
	logger "github.com/linkit360/go-utils/log"
)

type Mobilink struct {
	conf        Config
	rps         int
	ThrottleMT  <-chan time.Time
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
	notifier    *amqp.Notifier
}

type Config struct {
	Rps            int                  `default:"10" yaml:"rps"`
	Enabled        bool                 `default:"true" yaml:"enabled"`
	MTChanCapacity int                  `default:"1000" yaml:"channel_capacity"`
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
	MT      MTConfig      `yaml:"mt" json:"mt"`
	Content ContentConfig `yaml:"smpp" json:"content"`
}

type ContentConfig struct {
	Endpoint string `default:"http://52.29.238.205:4444/cgi-bin/sendsms" yaml:"endpoint"`
	User     string `default:"user" yaml:"user"`
	Password string `default:"password" yaml:"pass"`
	From     string `default:"Slypee" yaml:"from"`
	SMSC     string `default:"SLYEPPLA" yaml:"smsc"`
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

// prefix from table
func My(msisdn string) bool {
	return msisdn[:2] == "92"
}

func Init(
	mobilinkRps int,
	mobilinkConf Config,
	notifier *amqp.Notifier,
) *Mobilink {

	mb := &Mobilink{
		rps:  mobilinkRps,
		conf: mobilinkConf,
	}

	mb.notifier = notifier

	mb.ThrottleMT = time.Tick(time.Second / time.Duration(mobilinkConf.Rps+1))
	mb.requestLog = logger.GetFileLogger(mobilinkConf.TransactionLog.RequestLogPath)

	mb.responseLog = logger.GetFileLogger(mobilinkConf.TransactionLog.ResponseLogPath)

	var err error
	mb.location, err = time.LoadLocation(mobilinkConf.Location)
	if err != nil {
		log.WithFields(log.Fields{
			"location": mobilinkConf.Location,
			"error":    err,
		}).Fatal("init location")
	}

	mb.client = &http.Client{
		Timeout: time.Duration(mobilinkConf.Connection.MT.TimeoutSec) * time.Second,
	}
	log.Info("mobilink init done")

	return mb
}
