package mobilink

import (
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	smpp_client "github.com/fiorix/go-smpp/smpp"

	m "github.com/vostrok/operator/pk/mobilink/src/metrics"
	"github.com/vostrok/utils/amqp"
	logger "github.com/vostrok/utils/log"
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

	mb := &Mobilink{
		rps:  mobilinkRps,
		conf: mobilinkConf,
	}
	log.Info("mb metrics init done")

	mb.notifier = notifier

	mb.ThrottleMT = time.Tick(time.Second / time.Duration(mobilinkConf.Rps+1))
	mb.requestLog = logger.GetFileLogger(mobilinkConf.TransactionLog.RequestLogPath)
	log.Info("request logger init done")

	mb.responseLog = logger.GetFileLogger(mobilinkConf.TransactionLog.ResponseLogPath)
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
				m.SMPPConnected.Set(0)
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
				m.SMPPConnected.Set(1)
			}
		}
	}()
	return mb
}
