package config

import (
	"flag"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/configor"

	inmem_client "github.com/vostrok/inmem/rpcclient"
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
	MetricInstancePrefix string                       `yaml:"metric_instance_prefix"`
	AppName              string                       `yaml:"app_name"`
	Server               ServerConfig                 `yaml:"server"`
	DB                   db.DataBaseConfig            `yaml:"db"`
	Consumer             amqp.ConsumerConfig          `yaml:"consumer"`
	Publisher            amqp.NotifierConfig          `yaml:"publisher"`
	Yondo                YonduConfig                  `yaml:"yondu"`
	InMem                inmem_client.RPCClientConfig `yaml:"inmem"`
}

type YonduConfig struct {
	Name           string               `yaml:"name"`
	Auth           BasicAuth            `yaml:"auth"`
	Timeout        int                  `default:"30" yaml:"timeout"`
	APIUrl         string               `default:"http://localhost:50306/" yaml:"api_url"`
	TransactionLog TransactionLogConfig `yaml:"transaction_log"`
	ResponseCode   map[string]string    `yaml:"response_code"`
	Queue          YonduQueuesConfig    `yaml:"queues"`
	Tariffs        map[int]string       `yaml:"tariffs"`
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
	SendConsent    config.ConsumeQueueConfig `yaml:"consent"`
	Charge         config.ConsumeQueueConfig `yaml:"charge"`
	MT             config.ConsumeQueueConfig `yaml:"mt"`
	CallBack       config.ConsumeQueueConfig `yaml:"callback"`
	MO             config.ConsumeQueueConfig `yaml:"mo"`
	TransactionLog string                    `yaml:"transaction_log" default:"transaction_log"`
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

	if appConfig.AppName == "" {
		log.Fatal("app_name must be defiled as <host>_<name>")
	}
	if strings.Contains(appConfig.AppName, "-") {
		log.Fatal("app_name must be without '-' : it's not a valid metric name")
	}
	if appConfig.MetricInstancePrefix == "" {
		log.Fatal("metric_instance_prefix be defiled as <host>_<name>")
	}
	if strings.Contains(appConfig.MetricInstancePrefix, "-") {
		log.Fatal("metric_instance_prefix be without '-' : it's not a valid metric name")
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
