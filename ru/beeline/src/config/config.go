package config

import (
	"flag"
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/configor"

	inmem_client "github.com/linkit360/go-inmem/rpcclient"
	"github.com/linkit360/go-utils/amqp"
	"github.com/linkit360/go-utils/db"
)

type ServiceConfig struct {
	Beeline  BeelineConfig
	Consumer amqp.ConsumerConfig
	Notifier amqp.NotifierConfig
}
type ServerConfig struct {
	Port string `default:"50306"`
}
type AppConfig struct {
	AppName   string                    `yaml:"app_name"`
	Server    ServerConfig              `yaml:"server"`
	Beeline   BeelineConfig             `yaml:"beeline"`
	DB        db.DataBaseConfig         `yaml:"db"`
	Consumer  amqp.ConsumerConfig       `yaml:"consumer"`
	Publisher amqp.NotifierConfig       `yaml:"publisher"`
	InMem     inmem_client.ClientConfig `yaml:"inmem"`
}

type BeelineConfig struct {
	Name                   string               `yaml:"name"`
	MccMnc                 int64                `yaml:"mccmnc"`
	CountryCode            int64                `yaml:"country_code"`
	ShortNUmber            string               `yaml:"short_number"`
	TransactionLogFilePath TransactionLogConfig `yaml:"transaction_log"`
	Queue                  BeelineQueuesConfig  `yaml:"queues"`
	SMPP                   SmppConfig           `yaml:"smpp"`
}

type SmppConfig struct {
	Addr           string `yaml:"addr"`
	User           string `yaml:"user"`
	Password       string `yaml:"pass"`
	Timeout        int    `yaml:"timeout"`
	ReconnectDelay int    `yaml:"reconnect_delay"`
}

type ThrottleConfig struct {
	HTTP int `yaml:"http"`
}

type TransactionLogConfig struct {
	ResponseLogPath string `default:"/var/log/linkit/response_beeline.log" yaml:"response"`
	RequestLogPath  string `default:"/var/log/linkit/request_beeline.log" yaml:"request"`
}
type BeelineQueuesConfig struct {
	MO             string `yaml:"mo" default:"beeline_mo"`
	DBActions      string `yaml:"db_actions"`
	TransactionLog string `yaml:"transaction_log" default:"transaction_log"`
}

func LoadConfig() AppConfig {
	cfg := flag.String("config", "dev/beeline.yml", "configuration yml file")
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

	appConfig.Beeline.TransactionLogFilePath.ResponseLogPath =
		envString("RESPONSE_LOG", appConfig.Beeline.TransactionLogFilePath.ResponseLogPath)
	appConfig.Beeline.TransactionLogFilePath.RequestLogPath =
		envString("REQUEST_LOG", appConfig.Beeline.TransactionLogFilePath.RequestLogPath)

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
