package src

// Former corner for operator service
import (
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	//yondo "github.com/vostrok/operator/pk/yondo/src/api"
	"github.com/vostrok/operator/pk/yondo/src/config"
	"github.com/vostrok/operator/pk/yondo/src/service"
	m "github.com/vostrok/utils/metrics"
)

func RunServer() {
	appConfig := config.LoadConfig()
	m.Init(appConfig.Name)
	service.InitService(
		appConfig.Server,
		appConfig.Yondo,
		appConfig.Queues,
		appConfig.Consumer,
		appConfig.Publisher,
	)

	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	log.WithField("CPUCount", nuCPU)

	r := gin.New()
	//yondo.AddTestHandlers(r)
	m.AddHandler(r)

	r.Run(":" + appConfig.Server.Port)

	log.WithField("port", appConfig.Server.Port).Info("yondo init")
}
