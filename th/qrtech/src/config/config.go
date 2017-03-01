package config

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/configor"

	inmem_client "github.com/vostrok/inmem/rpcclient"
	"github.com/vostrok/utils/amqp"
	"github.com/vostrok/utils/db"
)

type ServiceConfig struct {
	Server   ServerConfig
	Consumer amqp.ConsumerConfig
	Notifier amqp.NotifierConfig
	QRTech   QRTechConfig
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
	QRTech    QRTechConfig                 `yaml:"qrtech"`
	InMem     inmem_client.RPCClientConfig `yaml:"inmem_client"`
}

type QRTechConfig struct {
	Name                   string               `yaml:"name"`
	AisMNC                 string               `yaml:"ais_mnc"`
	DtacMNC                string               `yaml:"dtac_mnc"`
	TruehMNC               string               `yaml:"trueh_mnc"`
	MCC                    string               `yaml:"mcc"`
	CountryCode            int64                `yaml:"country_code"`
	Location               string               `yaml:"location" default:"Asia/Bangkok"`
	InternalsPath          string               `yaml:"internals_path" default:"/home/centos/linkit/qrtech_internals.json"`
	MoToken                string               `yaml:"motoken"`
	TransactionLogFilePath TransactionLogConfig `yaml:"transaction_log"`
	Queue                  QRTechQueuesConfig   `yaml:"queues"`
	DN                     DNConfig             `yaml:"dn"`
	MT                     MTConfig             `yaml:"mt"`
}
type MTConfig struct {
	APIUrl     string            `default:"http://localhost:50306/" yaml:"url"`
	UserName   string            `default:"LinkIT360" yaml:"username"`
	Timeout    int               `default:"30" yaml:"timeout"`
	RPS        int               `yaml:"rps" default:"30"`
	ResultCode map[string]string `yaml:"result"`
}
type DNConfig struct {
	Code map[string]string `yaml:"code"`
}
type TransactionLogConfig struct {
	ResponseLogPath string `default:"/var/log/linkit/response_qrtech.log" yaml:"response"`
	RequestLogPath  string `default:"/var/log/linkit/request_qrtech.log" yaml:"request"`
}
type QRTechQueuesConfig struct {
	MO             string `yaml:"mo" default:"qrtech_mo"`
	DN             string `yaml:"dn" default:"qrtech_dn"`
	Unsubscribe    string `yaml:"unsubscribe" default:"mt_manager"`
	TransactionLog string `yaml:"transaction_log" default:"transaction_log"`
}
type InternalsConfig struct {
	sync.RWMutex
	MTLastAt map[int64]time.Time `json:"mt_last_at"`
}

func (ic *InternalsConfig) Load(path string) (err error) {
	ic.Lock()
	defer ic.Unlock()

	log.WithField("pid", os.Getpid()).Debug("load internals")

	fh, err := os.Open(path)
	if err != nil {
		log.WithField("error", err.Error()).Info("cannot open file")
		return
	}
	bufferBytes := bytes.NewBuffer(nil)
	_, err = io.Copy(bufferBytes, fh)
	if err != nil {
		log.WithField("error", err.Error()).Error("cannot copy from")
		return
	}
	if err = fh.Close(); err != nil {
		log.WithField("error", err.Error()).Error("cannot close fh")
		return
	}
	if err = json.Unmarshal(bufferBytes.Bytes(), ic); err != nil {
		log.WithField("error", err.Error()).Error("cannot unmarshal")
		return
	}
	log.Debug("read")
	return
}

func (ic *InternalsConfig) Save(path string) (err error) {
	ic.Lock()
	defer ic.Unlock()

	out, err := json.Marshal(ic)
	if err != nil {
		log.WithField("error", err.Error()).Error("cannot marshal")
		return
	}
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0744)
	if err != nil {
		log.WithField("error", err.Error()).Error("cannot open")
		return
	}
	_, err = fh.Write(out)
	if err != nil {
		log.WithField("error", err.Error()).Error("cannot write")
		return
	}
	if err = fh.Close(); err != nil {
		log.WithField("error", err.Error()).Error("cannot close")
		return
	}
	return nil
}

func LoadConfig() AppConfig {
	cfg := flag.String("config", "dev/qrtech.yml", "configuration yml file")
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

	appConfig.QRTech.TransactionLogFilePath.ResponseLogPath =
		envString("RESPONSE_LOG", appConfig.QRTech.TransactionLogFilePath.ResponseLogPath)
	appConfig.QRTech.TransactionLogFilePath.RequestLogPath =
		envString("REQUEST_LOG", appConfig.QRTech.TransactionLogFilePath.RequestLogPath)

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
