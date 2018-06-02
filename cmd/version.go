package cmd

import (
	"fmt"

	"github.com/ooni/collector/collector/info"
	"github.com/spf13/cobra"
)

// VersionCmd is the command used to output the version of orchestra
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of OONI Collector",
	Long:  `All software has versions. This is OONI Collector'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(info.FullVersionString())
		return nil
	},
}
