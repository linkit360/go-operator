package src

import (
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	"github.com/linkit360/go-operator/ru/beeline/src/config"
	"github.com/linkit360/go-operator/ru/beeline/src/metrics"
	"github.com/linkit360/go-operator/ru/beeline/src/service"
	m "github.com/linkit360/go-utils/metrics"
)

func RunServer() {
	appConfig := config.LoadConfig()

	metrics.Init(appConfig.AppName)

	service.InitService(
		appConfig.Beeline,
		appConfig.InMem,
		appConfig.Consumer,
		appConfig.Publisher,
	)

	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	log.WithField("CPUCount", nuCPU)

	r := gin.New()
	m.AddHandler(r)

	r.NoRoute(notFound)

	r.Run(":" + appConfig.Server.Port)

	log.WithField("port", appConfig.Server.Port).Info("beeline init")
}

func notFound(c *gin.Context) {
	log.WithFields(log.Fields{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"req":    c.Request.URL.RawQuery,
	}).Info("404notfound")
	metrics.PageNotFound.Inc()
	c.JSON(404, struct{}{})
}

func OnExit() {
	service.OnExit()
}
