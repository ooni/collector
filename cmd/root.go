package cmd

import (
	"fmt"
	"os"
	"strings"

	apexLog "github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var log = apexLog.WithFields(apexLog.Fields{
	"pkg": "cmd",
	"cmd": "ooni-registry",
})

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ooni-collector",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().Bool("dev", false, "run in development mode")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./ooni-collector.toml)")
	RootCmd.PersistentFlags().StringP("log-level", "", "info", "Set the log level")
	viper.BindPFlag("core.log-level", RootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("core.is-dev", RootCmd.PersistentFlags().Lookup("dev"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigName("ooni-collector")
	viper.AddConfigPath("/etc/ooni-collector/")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	replacer := strings.NewReplacer("-", "_", ".", "_") // Allows us to defined keys with - & ., but set them in via env variables with _
	viper.SetEnvKeyReplacer(replacer)

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	if err := viper.ReadInConfig(); err == nil {
		log.Infof("using config file: %s", viper.ConfigFileUsed())
	} else {
		log.WithError(err).Errorf("using default configuration")
	}

	apexLog.SetHandler(cli.Default)
	level, err := apexLog.ParseLevel(viper.GetString("core.log-level"))
	if err != nil {
		fmt.Println("Invalid log level. Must be one of debug, info, warn, error, fatal")
	}
	apexLog.SetLevel(level)
}
