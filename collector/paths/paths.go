package paths

import (
	"fmt"
	"path/filepath"

	"github.com/ooni/collector/collector/report"
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

// ClosedReportPath is the final path of a report. The filename looks like this:
// 20180601T172750Z-ndt-20180601T172754Z_AS14080_iR5R39aBde9hAcE6kMw7rOCAF0iR63IPSGtcMWYj0QDHHujaXu-AS14080-CO-probe-0.2.0.json
func ClosedReportPath(meta *report.Metadata) string {
	return filepath.Join(ReportDir(), fmt.Sprintf(
		"%s-%s-%s-%s-%s-probe-0.2.0.json",
		meta.CreationTime.Format(report.TimestampFormat),
		meta.TestName,
		meta.ReportID,
		meta.ProbeASN,
		meta.ProbeCC,
	))
}
