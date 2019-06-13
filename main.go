package main

import (
	"github.com/ooni/collector/cmd"
	"github.com/ooni/collector/collector"
)

var (
	version = "dev"
	commit  = ""
	date    = "unknown"
)

func main() {
	collector.Version = version
	collector.CommitHash = commit
	collector.BuildDate = date
	cmd.Execute()
}
