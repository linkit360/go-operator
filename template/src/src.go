package src

// Former corner for operator service
import (
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/contrib/expvar"
	"github.com/gin-gonic/gin"

	"github.com/vostrok/operator/pk/mobilink"
)

func RunServer() {
	appConfig := config.LoadConfig()
	service.InitService(appConfig.Server, appConfig.DbConf, appConfig.Consumer)

	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	log.WithField("CPUCount", nuCPU)

	r := gin.New()

	rg := r.Group("/debug")
	rg.GET("/vars", expvar.Handler())

	r.Run(":" + appConfig.Server.Port)

	log.WithField("port", appConfig.Server.Port).Info("mobilink init")
}
