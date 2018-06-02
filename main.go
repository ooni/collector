package main

import (
	"github.com/ooni/collector/cmd"
	"github.com/ooni/collector/collector/info"
)

var (
	version = "dev"
	commit  = ""
	date    = "unknown"
)

func main() {
	info.Version = version
	info.CommitHash = commit
	info.BuildDate = date
	cmd.Execute()
}
