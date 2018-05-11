package cmd

import (
	"github.com/ooni/collector/collector"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the collector service",
	Long:  `This is the main entrypoint for starting the collector service`,
	Run: func(cmd *cobra.Command, args []string) {
		collector.Start()
	},
}

func init() {
	RootCmd.AddCommand(startCmd)

	startCmd.PersistentFlags().IntP("port", "", 8080, "Which port we should bind to")
	startCmd.PersistentFlags().StringP("address", "", "127.0.0.1", "Which interface we should listen on")
	viper.BindPFlag("api.port", startCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("api.address", startCmd.PersistentFlags().Lookup("address"))
}
