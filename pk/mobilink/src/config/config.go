package config

import (
	"flag"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/configor"

	mobilink_api "github.com/vostrok/operator/pk/mobilink/src/api"
	"github.com/vostrok/utils/amqp"
)

type ServerConfig struct {
	Port         string `default:"50304"`
	OperatorName string `yaml:"operator_name"`
	ThreadsCount int    `default:"1" yaml:"threads_count"`
}
type QueueConfig struct {
	In  string `yaml:"-"`
	Out string `yaml:"-"`
}
type AppConfig struct {
	Name      string                `yaml:"name"`
	Server    ServerConfig          `yaml:"server"`
	Mobilink  mobilink_api.Config   `yaml:"mobilink"`
	Consumer  rabbit.ConsumerConfig `yaml:"consumer"`
	Publisher rabbit.NotifierConfig `yaml:"publisher"`
	Queues    QueueConfig           `yaml:"-"`
}

func LoadConfig() AppConfig {
	cfg := flag.String("config", "dev/appconfig.yml", "configuration yml file")
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

	appConfig.Queues.In = appConfig.Server.OperatorName + "_requests"
	appConfig.Queues.Out = appConfig.Server.OperatorName + "_responses"

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
