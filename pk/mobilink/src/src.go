package src

// Former corner for operator service
import (
	"runtime"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

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
	service.AddMobilinkTestHandlers(r)
	metrics.AddHandler(r)

	r.Run(appConfig.Server.Host + ":" + appConfig.Server.Port)
	log.WithField("dsn", appConfig.Server.Host+":"+appConfig.Server.Port).Info("mt init")
}
