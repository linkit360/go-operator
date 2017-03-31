package src

import (
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	"github.com/linkit360/go-operator/th/qrtech/src/config"
	"github.com/linkit360/go-operator/th/qrtech/src/metrics"
	"github.com/linkit360/go-operator/th/qrtech/src/service"
	m "github.com/linkit360/go-utils/metrics"
)

func RunServer() {
	appConfig := config.LoadConfig()

	metrics.Init(appConfig.AppName)

	service.InitService(
		appConfig.Server,
		appConfig.QRTech,
		appConfig.Consumer,
		appConfig.Publisher,
		appConfig.InMem,
	)

	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	log.WithField("CPUCount", nuCPU)

	e := gin.New()
	service.AddTestMTHandler(e)
	service.AddMOHandler(e)
	service.AddDNHandler(e)
	m.AddHandler(e)

	e.NoRoute(notFound)

	e.Run(":" + appConfig.Server.Port)

	log.WithField("port", appConfig.Server.Port).Info("qrTech init")
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
