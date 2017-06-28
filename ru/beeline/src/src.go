package src

import (
	"runtime"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/linkit360/go-operator/ru/beeline/src/config"
	"github.com/linkit360/go-operator/ru/beeline/src/metrics"
	"github.com/linkit360/go-operator/ru/beeline/src/service"
	m "github.com/linkit360/go-utils/metrics"
)

func RunServer() {
	appConfig := config.LoadConfig()

	metrics.Init(appConfig.AppName)

	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	log.WithField("CPUCount", nuCPU)

	r := gin.New()
	m.AddHandler(r)

	r.NoRoute(notFound)

	r.Run(appConfig.Server.Host + ":" + appConfig.Server.Port)
	log.WithField("dsn", appConfig.Server.Host+":"+appConfig.Server.Port).Info("mt init")

	service.InitService(
		appConfig.Beeline,
		appConfig.Mid,
		appConfig.Consumer,
		appConfig.Publisher,
	)
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
