package config

import (
	"flag"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/configor"

	"github.com/vostrok/utils/amqp"
	"github.com/vostrok/utils/config"
	"github.com/vostrok/utils/db"
)

type ServiceConfig struct {
	Server   ServerConfig
	Consumer amqp.ConsumerConfig
	Notifier amqp.NotifierConfig
	Yondu    YonduConfig
}
type ServerConfig struct {
	Port string `default:"50306"`
}
type AppConfig struct {
	Name      string              `yaml:"name"`
	Server    ServerConfig        `yaml:"server"`
	DB        db.DataBaseConfig   `yaml:"db"`
	Consumer  amqp.ConsumerConfig `yaml:"consumer"`
	Publisher amqp.NotifierConfig `yaml:"publisher"`
	Yondo     YonduConfig         `yaml:"yondu"`
}

type YonduConfig struct {
	Auth           BasicAuth            `yaml:"auth"`
	Timeout        int                  `default:"30" yaml:"timeout"`
	APIUrl         string               `default:"http://localhost:50306/" yaml:"api_url"`
	TransactionLog TransactionLogConfig `yaml:"transaction_log"`
	ResponseCode   map[int]string       `yaml:"response_code"`
	Queue          YonduQueuesConfig    `yaml:"queues"`
}
type BasicAuth struct {
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}
type TransactionLogConfig struct {
	ResponseLogPath string `default:"/var/log/linkit/yondu/response.log" yaml:"response"`
	RequestLogPath  string `default:"/var/log/linkit/yondu/request.log" yaml:"request"`
}
type YonduQueuesConfig struct {
	SendConsent     config.ConsumeQueueConfig `yaml:"send_consent"`
	VerifyTransCode config.ConsumeQueueConfig `yaml:"verify_trans_code"`
	Charge          config.ConsumeQueueConfig `yaml:"yondu_charge"` // name yondu_requests
	MT              config.ConsumeQueueConfig `yaml:"yondu_mt"`
	TransactionLog  string                    `yaml:"transaction_log"`
	CallBack        string                    `yaml:"callback" default:"yondu_responses"`
	MO              string                    `yaml:"mo" default:"yondu_sms_responses"`
}

func LoadConfig() AppConfig {
	cfg := flag.String("config", "dev/yondo.yml", "configuration yml file")
	flag.Parse()
	var appConfig AppConfig

	if *cfg != "" {
		if err := configor.Load(&appConfig, *cfg); err != nil {
			log.WithField("config", err.Error()).Fatal("config load error")
		}
	}

	if appConfig.Name == "" {
		log.Fatal("app name must be defiled as <host>_<name>")
	}
	if strings.Contains(appConfig.Name, "-") {
		log.Fatal("app name must be without '-' : it's not a valid metric name")
	}

	appConfig.Server.Port = envString("PORT", appConfig.Server.Port)
	appConfig.Consumer.Conn.Host = envString("RBMQ_HOST", appConfig.Consumer.Conn.Host)
	appConfig.Publisher.Conn.Host = envString("RBMQ_HOST", appConfig.Publisher.Conn.Host)

	appConfig.Yondo.TransactionLog.ResponseLogPath =
		envString("RESPONSE_LOG", appConfig.Yondo.TransactionLog.ResponseLogPath)
	appConfig.Yondo.TransactionLog.RequestLogPath =
		envString("REQUEST_LOG", appConfig.Yondo.TransactionLog.RequestLogPath)

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
