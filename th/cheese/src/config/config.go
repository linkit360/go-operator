package config

import (
	"flag"
	"fmt"
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
	Cheese   CheeseConfig
}
type ServerConfig struct {
	Port string `default:"50306"`
}
type AppConfig struct {
	AppName   string                       `yaml:"app_name"`
	Server    ServerConfig                 `yaml:"server"`
	DB        db.DataBaseConfig            `yaml:"db"`
	Consumer  amqp.ConsumerConfig          `yaml:"consumer"`
	Publisher amqp.NotifierConfig          `yaml:"publisher"`
	Cheese    CheeseConfig                 `yaml:"cheese"`
	InMem     inmem_client.RPCClientConfig `yaml:"inmem"`
}

type CheeseConfig struct {
	Name                   string               `yaml:"name"`
	Timeout                int                  `default:"30" yaml:"timeout"`
	APIUrl                 string               `default:"http://localhost:50306/" yaml:"api_url"`
	Throttle               ThrottleConfig       `yaml:"throttle,omitempty"`
	TransactionLogFilePath TransactionLogConfig `yaml:"transaction_log"`
	Queue                  CheeseQueuesConfig   `yaml:"queues"`
}
type ThrottleConfig struct {
}
type TransactionLogConfig struct {
	ResponseLogPath string `default:"/var/log/linkit/response_cheese.log" yaml:"response"`
	RequestLogPath  string `default:"/var/log/linkit/request_cheese.log" yaml:"request"`
}
type CheeseQueuesConfig struct {
	Ais            config.ConsumeQueueConfig `yaml:"ais"`
	Dtac           config.ConsumeQueueConfig `yaml:"dtac"`
	Trueh          config.ConsumeQueueConfig `yaml:"trueh"`
	TransactionLog string                    `yaml:"transaction_log" default:"transaction_log"`
}

func LoadConfig() AppConfig {
	cfg := flag.String("config", "dev/cheese.yml", "configuration yml file")
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
	appConfig.Server.Port = envString("PORT", appConfig.Server.Port)
	appConfig.Consumer.Conn.Host = envString("RBMQ_HOST", appConfig.Consumer.Conn.Host)
	appConfig.Publisher.Conn.Host = envString("RBMQ_HOST", appConfig.Publisher.Conn.Host)

	appConfig.Cheese.TransactionLogFilePath.ResponseLogPath =
		envString("RESPONSE_LOG", appConfig.Cheese.TransactionLogFilePath.ResponseLogPath)
	appConfig.Cheese.TransactionLogFilePath.RequestLogPath =
		envString("REQUEST_LOG", appConfig.Cheese.TransactionLogFilePath.RequestLogPath)

	log.WithField("config", fmt.Sprintf("%#v", appConfig)).Info("Config loaded")
	return appConfig
}

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}
