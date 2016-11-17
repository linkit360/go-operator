package src

// Former corner for operator service
import (
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	mobilink_api "github.com/vostrok/operator/pk/mobilink/src/api"
	"github.com/vostrok/operator/pk/mobilink/src/config"
	"github.com/vostrok/operator/pk/mobilink/src/service"
)

func RunServer() {
	appConfig := config.LoadConfig()
	service.InitService(
		appConfig.Server,
		appConfig.Mobilink,
		appConfig.DbConf,
		appConfig.Queues,
		appConfig.Consumer,
		appConfig.Publisher,
	)

	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	log.WithField("CPUCount", nuCPU)

	r := gin.New()
	rgMobilink := r.Group("/mobilink_handler")
	rgMobilink.POST("", mobilink_api.MobilinkHandler)

	r.Run(":" + appConfig.Server.Port)

	log.WithField("port", appConfig.Server.Port).Info("mobilink init")
}
