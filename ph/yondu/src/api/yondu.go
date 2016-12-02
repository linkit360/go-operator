package yondo

import (
	"encoding/base64"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"

	//m "github.com/vostrok/operator/ph/yondu/src/metrics"
	inmem_client "github.com/vostrok/inmem/rpcclient"
	"github.com/vostrok/utils/amqp"
	logger "github.com/vostrok/utils/log"
)

type Config struct {
	Auth           BasicAuth            `yaml:"auth"`
	Timeout        int                  `default:"30" yaml:"timeout"`
	APIUrl         string               `default:"http://localhost:50306/" yaml:"api_url"`
	TransactionLog TransactionLogConfig `yaml:"log_transaction"`
	ResponseCode   map[int]string       `yaml:"response_code"`
}
type BasicAuth struct {
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}
type Yondu struct {
	conf        Config
	notifier    *amqp.Notifier
	ThrottleMT  <-chan time.Time
	location    *time.Location
	client      *http.Client
	responseLog *log.Logger
	requestLog  *log.Logger
}

type TransactionLogConfig struct {
	Queue           string `default:"transaction_log" yaml:"transaction_log"`
	ResponseLogPath string `default:"/var/log/yondu/response.log" yaml:"response"`
	RequestLogPath  string `default:"/var/log/yondu/request.log" yaml:"request"`
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func Init(yConf Config, inMemConfig inmem_client.RPCClientConfig) *Yondu {
	if err := inmem_client.Init(inMemConfig, inmem_client.InitMetrics()); err != nil {
		log.WithField("error", err.Error()).Fatal("cannot init inmem client")
	}
	y := Yondu{
		conf:        yConf,
		responseLog: logger.GetFileLogger(yConf.TransactionLog.ResponseLogPath),
		requestLog:  logger.GetFileLogger(yConf.TransactionLog.RequestLogPath),
	}
	return y
}
