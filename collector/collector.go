package collector

import (
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// SetupRouter will create a *gin.Engine
func SetupRouter(reportDir string) *gin.Engine {
	store, err := NewStorage(reportDir)
	if err != nil {
		log.WithError(err).Error("failed to init storage")
		return nil
	}

	storageMw, err := InitStorageMiddleware(store)
	if err != nil {
		log.WithError(err).Error("failed to init storage middleware")
		return nil
	}

	router := gin.Default()
	router.Use(storageMw.MiddlewareFunc())
	err = BindAPI(router)
	if err != nil {
		log.WithError(err).Error("failed to BindAPI")
		return nil
	}
	return router
}

// Start the collector server
func Start() {
	var (
		err error
	)
	if viper.GetBool("core.is-dev") != true {
		gin.SetMode(gin.ReleaseMode)
	}

	router := SetupRouter(viper.GetString("core.report-dir"))
	if router == nil {
		log.Error("failed to setup router")
		return
	}

	Addr := fmt.Sprintf("%s:%d", viper.GetString("api.address"),
		viper.GetInt("api.port"))
	log.Infof("starting on %s", Addr)

	servers := []*http.Server{
		&http.Server{
			Addr:    Addr,
			Handler: router,
		},
	}
	err = gracehttp.ServeWithOptions(servers, nil)
	if err != nil {
		log.WithError(err).Error("failed to start server")
	}
}
