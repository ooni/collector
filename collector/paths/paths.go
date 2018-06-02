package paths

import (
	"path/filepath"

	"github.com/spf13/viper"
)

// ReportDir is the path to use for storing completed reports
func ReportDir() string {
	return filepath.Join(viper.GetString("core.data-root"), "reports")
}

// TempReportDir is the path to use for storing temporary reports
func TempReportDir() string {
	return filepath.Join(viper.GetString("core.data-root"), "temp-reports")
}

// BadgerDir is the path to the badger database
func BadgerDir() string {
	return filepath.Join(viper.GetString("core.data-root"), "badger")
}
