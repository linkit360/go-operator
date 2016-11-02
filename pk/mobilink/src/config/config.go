package config

import (
	"flag"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/configor"

	"github.com/vostrok/db"
	"github.com/vostrok/rabbit"
)

type ServerConfig struct {
	Port         string `default:"50304"`
	Queue        string `default:"mobilink" yaml:"queue"`
	ThreadsCount int    `default:"1" yaml:"threads_count"`
}
type NewRelicConfig struct {
	AppName string `default:"mimbai.mobilink.linkit360.com"`
	License string `default:"4d635427ad90ca786ca2db6aa246ed651730b933"`
}

type AppConfig struct {
	Server   ServerConfig          `yaml:"server"`
	NewRelic NewRelicConfig        `yaml:"newrelic"`
	DbConf   db.DataBaseConfig     `yaml:"db"`
	Consumer rabbit.ConsumerConfig `yaml:"consumer"`
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

	appConfig.Server.Port = envString("PORT", appConfig.Server.Port)
	appConfig.Consumer.Uri = envString("RBMQ_URL", appConfig.Consumer.Uri)

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
