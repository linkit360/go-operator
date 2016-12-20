package config

import (
	"flag"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/configor"

	mobilink_api "github.com/vostrok/operator/pk/mobilink/src/api"
	"github.com/vostrok/utils/amqp"
	"github.com/vostrok/utils/config"
)

type ServerConfig struct {
	Port         string `default:"50304"`
	OperatorName string `yaml:"operator_name"`
	ThreadsCount int    `default:"1" yaml:"threads_count"`
}

type QueueConfig struct {
	Requests    config.ConsumeQueueConfig `yaml:"requests"`
	SMSRequests config.ConsumeQueueConfig `yaml:"sms_requests"`
}

type AppConfig struct {
	MetricInstancePrefix string              `yaml:"metric_instance_prefix"`
	AppName              string              `yaml:"app_name"`
	Server               ServerConfig        `yaml:"server"`
	Queues               QueueConfig         `yaml:"queues"`
	Consumer             amqp.ConsumerConfig `yaml:"consumer"`
	Publisher            amqp.NotifierConfig `yaml:"publisher"`
	Mobilink             mobilink_api.Config `yaml:"mobilink"`
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
	if appConfig.MetricInstancePrefix == "" {
		log.Fatal("metric_instance_prefix be defiled as <host>_<name>")
	}
	if strings.Contains(appConfig.MetricInstancePrefix, "-") {
		log.Fatal("metric_instance_prefix be without '-' : it's not a valid metric name")
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
