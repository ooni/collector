package cmd

import (
	"fmt"
	"net/http"

	"github.com/ooni/collector/collector/api/v1"
	"github.com/ooni/collector/collector/aws"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	apexLog "github.com/apex/log"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var log = apexLog.WithFields(apexLog.Fields{
	"pkg": "collector",
	"cmd": "ooni-collector",
})

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

	if err = initAWS(); err != nil {
		log.WithError(err).Error("failed to init aws")
	}

	dbURL := viper.GetString("db.url")
	store, err := NewStorage(dbURL)
	if err != nil {
		log.WithError(err).Error("failed to create Storage")
		return
	}

	storageMw, err := InitStorageMiddleware(store)
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
	viper.SetDefault("api.admin-password", "changeme")
	viper.SetDefault("api.fqn", "unknown")
	viper.SetDefault("db.url", "")
	viper.SetDefault("aws.access-key-id", "")
	viper.SetDefault("aws.secret-access-key", "")
	viper.SetDefault("aws.s3-bucket", "ooni-collector")
	viper.SetDefault("aws.s3-prefix", "reports")
}
