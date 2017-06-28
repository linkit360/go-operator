package src

import (
	"runtime"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

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
		appConfig.Mid,
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

	r.Run(appConfig.Server.Host + ":" + appConfig.Server.Port)
	log.WithField("dsn", appConfig.Server.Host+":"+appConfig.Server.Port).Info("mt init")
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
