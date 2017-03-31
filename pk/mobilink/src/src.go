package src

// Former corner for operator service
import (
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	mobilink_api "github.com/linkit360/go-operator/pk/mobilink/src/api"
	"github.com/linkit360/go-operator/pk/mobilink/src/config"
	m "github.com/linkit360/go-operator/pk/mobilink/src/metrics"
	"github.com/linkit360/go-operator/pk/mobilink/src/service"
	metrics "github.com/linkit360/go-utils/metrics"
)

func RunServer() {
	appConfig := config.LoadConfig()
	m.Init(appConfig.AppName)

	service.InitService(
		appConfig.Server,
		appConfig.Mobilink,
		appConfig.Queues,
		appConfig.Consumer,
		appConfig.Publisher,
	)

	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	log.WithField("CPUCount", nuCPU)

	r := gin.New()
	mobilink_api.AddMobilinkTestHandlers(r)
	metrics.AddHandler(r)

	r.Run(":" + appConfig.Server.Port)

	log.WithField("port", appConfig.Server.Port).Info("mobilink init")
}
