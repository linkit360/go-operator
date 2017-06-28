package config

import (
	"flag"
	"os"
	"strings"

	"github.com/jinzhu/configor"
	log "github.com/sirupsen/logrus"

	"github.com/linkit360/go-utils/amqp"
	"github.com/linkit360/go-utils/config"
)

type ServerConfig struct {
	Host         string `default:"127.0.0.1" yaml:"host"`
	Port         string `default:"50304"`
	OperatorName string `yaml:"operator_name"`
}

type QueueConfig struct {
	Requests               config.ConsumeQueueConfig `yaml:"requests"`
	SMSRequests            config.ConsumeQueueConfig `yaml:"sms_requests"`
	OperatorTransactionLog string                    `yaml:"transaction_log"`
}

type AppConfig struct {
	AppName   string              `yaml:"app_name"`
	Server    ServerConfig        `yaml:"server"`
	Queues    QueueConfig         `yaml:"queues"`
	Consumer  amqp.ConsumerConfig `yaml:"consumer"`
	Publisher amqp.NotifierConfig `yaml:"publisher"`
	Mobilink  MobilinkConfig      `yaml:"mobilink"`
}

type MobilinkConfig struct {
	Rps            int                  `default:"10" yaml:"rps"`
	Enabled        bool                 `default:"true" yaml:"enabled"`
	MTChanCapacity int                  `default:"1000" yaml:"channel_capacity"`
	Location       string               `default:"Asia/Karachi" yaml:"location"`
	TransactionLog TransactionLogConfig `yaml:"log_transaction"`
	MT             MTConfig             `yaml:"mt" json:"mt"`
	Content        ContentConfig        `yaml:"content" json:"content"`
}
type TransactionLogConfig struct {
	Queue           string `default:"transaction_log" yaml:"transaction_log"`
	ResponseLogPath string `default:"/var/log/response_mobilink.log" yaml:"response"`
	RequestLogPath  string `default:"/var/log/request_mobilink.log" yaml:"request"`
}

type ContentConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"`
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

func LoadConfig() AppConfig {
	cfg := flag.String("config", "dev/mobilink.yml", "configuration yml file")
	flag.Parse()
	var appConfig AppConfig

	if *cfg != "" {
		if err := configor.Load(&appConfig, *cfg); err != nil {
			log.WithField("config", err.Error()).Fatal("config load error")
		}
	}

	if appConfig.AppName == "" {
		log.Fatal("app name must be defiled as <host>_<name>")
	}
	if strings.Contains(appConfig.AppName, "-") {
		log.Fatal("app name must be without '-' : it's not a valid metric name")
	}
	appConfig.Server.Port = envString("PORT", appConfig.Server.Port)
	appConfig.Consumer.Conn.Host = envString("RBMQ_HOST", appConfig.Consumer.Conn.Host)
	appConfig.Publisher.Conn.Host = envString("RBMQ_HOST", appConfig.Publisher.Conn.Host)

	appConfig.Mobilink.TransactionLog.ResponseLogPath =
		envString("MOBILINK_RESPONSE_LOG", appConfig.Mobilink.TransactionLog.ResponseLogPath)
	appConfig.Mobilink.TransactionLog.RequestLogPath =
		envString("MOBILINK_REQUEST_LOG", appConfig.Mobilink.TransactionLog.RequestLogPath)

	log.WithField("config", appConfig).Info("Config loaded")
	return appConfig
}

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}
