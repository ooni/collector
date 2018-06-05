package collector

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ooni/collector/collector/api/v1"
	"github.com/ooni/collector/collector/aws"
	"github.com/ooni/collector/collector/middleware"
	"github.com/ooni/collector/collector/paths"
	"github.com/ooni/collector/collector/report"
	"github.com/ooni/collector/collector/storage"

	apexLog "github.com/apex/log"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var log = apexLog.WithFields(apexLog.Fields{
	"pkg": "collector",
	"cmd": "ooni-collector",
})

func initDataRoot() error {
	requiredDirs := []string{
		paths.ReportDir(),
		paths.TempReportDir(),
		paths.BadgerDir(),
	}
	for _, path := range requiredDirs {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := os.Mkdir(path, 0700)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func initAWS() error {
	accessKeyID := viper.GetString("aws.access-key-id")
	secretAccessKey := viper.GetString("aws.secret-access-key")
	if accessKeyID == "" {
		return nil
	}
	aws.Session = aws.NewSession(accessKeyID, secretAccessKey)
	return nil
}

// Start the collector server
func Start() {
	var (
		err error
	)
	if viper.GetBool("core.is-dev") != true {
		gin.SetMode(gin.ReleaseMode)
	}

	adminPassword := viper.GetString("api.admin-password")
	if adminPassword == "changeme" {
		log.Warn("api.admin-password is set to the default value")
	}

	if err = initDataRoot(); err != nil {
		log.WithError(err).Error("failed to init data root")
	}
	if err = initAWS(); err != nil {
		log.WithError(err).Error("failed to init aws")
	}

	store := storage.New(paths.BadgerDir())
	storageMw, err := middleware.InitStorageMiddleware(store)
	if err != nil {
		log.WithError(err).Error("failed to init storage middleware")
		return
	}

	router := gin.Default()
	router.Use(storageMw.MiddlewareFunc())
	err = apiv1.BindAPI(router)
	if err != nil {
		log.WithError(err).Error("failed to BindAPI")
		return
	}
	report.ReloadExpiryTimers(store)

	Addr := fmt.Sprintf("%s:%d", viper.GetString("api.address"),
		viper.GetInt("api.port"))
	log.Infof("starting on %s", Addr)

	servers := []*http.Server{
		&http.Server{
			Addr:    Addr,
			Handler: router,
		},
	}
	opt := gracehttp.PreStartProcess(func() error {
		return store.Close()
	})
	err = gracehttp.ServeWithOptions(servers, opt)
	if err != nil {
		log.WithError(err).Error("failed to start server")
	}
}
