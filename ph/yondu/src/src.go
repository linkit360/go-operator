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
	service.AddTestHandlers(r)
	service.AddHandlers(r)
	m.AddHandler(r)

	r.NoRoute(notFound)

	r.Run(":" + appConfig.Server.Port)

	log.WithField("port", appConfig.Server.Port).Info("yondo init")
}

func notFound(c *gin.Context) {
	log.WithFields(log.Fields{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"req":    c.Request.URL.RawQuery,
	}).Info("404notfound")
	metrics.PageNotFound.Inc()
	c.JSON(200, struct{}{})
}
