package collector

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ooni/collector/collector/api/v1"
	"github.com/ooni/collector/collector/handler"
	"github.com/ooni/collector/collector/middleware"
	"github.com/ooni/collector/collector/paths"
	"github.com/ooni/collector/collector/report"
	"github.com/ooni/collector/storage"

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

// Start the collector server
func Start() {
	var (
		err error
	)

	if err = initDataRoot(); err != nil {
		log.WithError(err).Error("failed to init data root")
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

	reportList, err := store.ListReports()
	if err != nil {
		log.WithError(err).Error("failed to list reports")
		return
	}

	for _, meta := range reportList {
		report.ExpiryTimers[meta.ReportID] = time.AfterFunc(time.Duration(8)*time.Hour, func() {
			handler.CloseReport(store, meta.ReportID)
		})
	}

	Addr := fmt.Sprintf("%s:%d", viper.GetString("api.address"),
		viper.GetInt("api.port"))
	log.Infof("starting on %s", Addr)

	s := &http.Server{
		Addr:    Addr,
		Handler: router,
	}
	gracehttp.Serve(s)
}
