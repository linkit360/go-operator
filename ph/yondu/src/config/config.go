package config

import (
	"flag"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/configor"

	yondo "github.com/vostrok/operator/ph/yondu/src/api"
	"github.com/vostrok/utils/amqp"
	"github.com/vostrok/utils/config"
)

type ServerConfig struct {
	Port         string `default:"50304"`
	OperatorName string `yaml:"operator_name"`
	ThreadsCount int    `default:"1" yaml:"threads_count"`
}
type AppConfig struct {
	Name      string                     `yaml:"name"`
	Server    ServerConfig               `yaml:"server"`
	Consumer  amqp.ConsumerConfig        `yaml:"consumer"`
	Publisher amqp.NotifierConfig        `yaml:"publisher"`
	Yondo     yondo.Config               `yaml:"yondu"`
	Queues    config.OperatorQueueConfig `yaml:"-"`
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

	operator := envString("OPERATOR", "yondo")
	operators := make(map[string]config.OperatorConfig, 1)
	operators[operator] = config.OperatorConfig{}
	appConfig.Queues = config.GetOperatorsQueue(operators)[operator]

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
