package config

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jinzhu/configor"
	log "github.com/sirupsen/logrus"

	mid_client "github.com/linkit360/go-mid/rpcclient"
	"github.com/linkit360/go-utils/amqp"
	"github.com/linkit360/go-utils/config"
	"github.com/linkit360/go-utils/db"
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
	AppName   string                  `yaml:"app_name"`
	Server    ServerConfig            `yaml:"server"`
	Yondu     YonduConfig             `yaml:"yondu"`
	DB        db.DataBaseConfig       `yaml:"db"`
	Consumer  amqp.ConsumerConfig     `yaml:"consumer"`
	Publisher amqp.NotifierConfig     `yaml:"publisher"`
	Mid       mid_client.ClientConfig `yaml:"mid_client"`
}

type YonduConfig struct {
	Name                   string               `yaml:"name"`
	AuthKey                string               `yaml:"auth_key"`
	APIUrl                 string               `default:"http://localhost:50306/" yaml:"api_url"`
	Timeout                int                  `default:"30" yaml:"timeout"`
	Throttle               ThrottleConfig       `yaml:"throttle"`
	TransactionLogFilePath TransactionLogConfig `yaml:"transaction_log"`
	MTResponseCode         map[string]string    `yaml:"mt_code"`
	DNCode                 map[string]string    `yaml:"dn_code"`
	TariffCode             map[string]string    `yaml:"tariff"`
	Queue                  YonduQueuesConfig    `yaml:"queues"`
}
type ThrottleConfig struct {
	MT int `yaml:"mt" default:"10"`
}
type TransactionLogConfig struct {
	ResponseLogPath string `default:"/var/log/linkit/yondu_response.log" yaml:"response"`
	RequestLogPath  string `default:"/var/log/linkit/yondu_request.log" yaml:"request"`
}
type YonduQueuesConfig struct {
	MO             string                    `yaml:"mo"`
	DN             string                    `yaml:"dn"`
	TransactionLog string                    `yaml:"transaction_log" default:"transaction_log"`
	MT             config.ConsumeQueueConfig `yaml:"mt"`
}

func LoadConfig() AppConfig {
	cfg := flag.String("config", "dev/yondu.yml", "configuration yml file")
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

	appConfig.Yondu.TransactionLogFilePath.ResponseLogPath =
		envString("RESPONSE_LOG", appConfig.Yondu.TransactionLogFilePath.ResponseLogPath)
	appConfig.Yondu.TransactionLogFilePath.RequestLogPath =
		envString("REQUEST_LOG", appConfig.Yondu.TransactionLogFilePath.RequestLogPath)

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
