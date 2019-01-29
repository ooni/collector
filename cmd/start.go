package cmd

import (
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/collector/collector"
	"github.com/ooni/collector/collector/api/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"
)

// Start the collector server
func Start() {
	var (
		err error
	)
	if viper.GetBool("core.is-dev") != true {
		gin.SetMode(gin.ReleaseMode)
	}

	reportDir := viper.GetString("core.report-dir")
	store, err := collector.NewStorage(reportDir)
	if err != nil {
		log.WithError(err).Error("failed to init storage")
		return
	}

	storageMw, err := collector.InitStorageMiddleware(store)
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

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the collector service",
	Long:  `This is the main entrypoint for starting the collector service`,
	Run: func(cmd *cobra.Command, args []string) {
		Start()
	},
}

func init() {
	RootCmd.AddCommand(startCmd)

	startCmd.PersistentFlags().IntP("port", "", 8080, "Which port we should bind to")
	startCmd.PersistentFlags().StringP("address", "", "127.0.0.1", "Which interface we should listen on")

	viper.BindPFlag("api.port", startCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("api.address", startCmd.PersistentFlags().Lookup("address"))
	viper.SetDefault("api.fqn", "unknown")
}
