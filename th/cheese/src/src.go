package src

import (
	"runtime"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/linkit360/go-operator/th/cheese/src/config"
	"github.com/linkit360/go-operator/th/cheese/src/metrics"
	"github.com/linkit360/go-operator/th/cheese/src/service"
	m "github.com/linkit360/go-utils/metrics"
)

func RunServer() {
	appConfig := config.LoadConfig()

	metrics.Init(appConfig.AppName)

	service.InitService(
		appConfig.Server,
		appConfig.Cheese,
		appConfig.Consumer,
		appConfig.Publisher,
		appConfig.Mid,
	)

	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	log.WithField("CPUCount", nuCPU)

	r := gin.New()
	service.AddTestHandlers(r)
	service.AddHandlers(r)
	m.AddHandler(r)

	r.NoRoute(notFound)

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
