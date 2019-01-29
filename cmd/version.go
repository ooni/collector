package cmd

import (
	"fmt"

	"github.com/ooni/collector/collector"
	"github.com/spf13/cobra"
)

// versionCmd is the command used to output the version of orchestra
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of OONI Collector",
	Long:  `All software has versions. This is OONI Collector'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(collector.FullVersionString())
		return nil
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
