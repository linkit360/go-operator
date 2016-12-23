package src

import (
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	"github.com/vostrok/operator/ph/yondu/src/config"
	"github.com/vostrok/operator/ph/yondu/src/metrics"
	"github.com/vostrok/operator/ph/yondu/src/service"
	m "github.com/vostrok/utils/metrics"
)

func RunServer() {
	appConfig := config.LoadConfig()

	m.Init(appConfig.MetricInstancePrefix)
	metrics.Init(appConfig.AppName)

	service.InitService(
		appConfig.Server,
		appConfig.Yondo,
		appConfig.Consumer,
		appConfig.Publisher,
		appConfig.InMem,
	)

	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	log.WithField("CPUCount", nuCPU)

	r := gin.New()
	//yondo.AddTestHandlers(r)
	m.AddHandler(r)

	r.Run(":" + appConfig.Server.Port)

	log.WithField("port", appConfig.Server.Port).Info("yondo init")
}
