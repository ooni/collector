package report

import (
	"fmt"
	"time"

	"github.com/ooni/collector/collector/util"
)

// ExpiryTimers is a map of timers keyed to the ReportID. These are used to
// ensure that after a certain amount of time has elapsed reports are closed
var ExpiryTimers = make(map[string]*time.Timer)

// ExpiryTimeDuration is after how much time a report expires
var ExpiryTimeDuration = time.Duration(8) * time.Hour

// BackendExtra is serverside extra metadata
type BackendExtra struct {
	SubmissionTime time.Time `json:"submission_time"`
}

// MeasurementEntry is the structure of measurements submitted by an OONI Probe client
type MeasurementEntry struct {
	// These values are added by the pipeline
	BucketDate     string `json:"bucket_date"`
	ReportFilename string `json:"report_filename"`

	ID                   string       `json:"id"`
	ReportID             string       `json:"report_id"`
	TestName             string       `json:"test_name"`
	TestVersion          string       `json:"test_version"`
	MeasurementStartTime string       `json:"measurement_start_time"`
	TestStartTime        string       `json:"test_start_time"` // XXX these should actually be time
	Annotations          interface{}  `json:"annotations"`
	BackendExtra         BackendExtra `json:"backend_extra"`
	BackendVersion       string       `json:"backend_version"`
	DataFormatVersion    string       `json:"data_format_version"`
	Input                string       `json:"input"`
	InputHashes          []string     `json:"input_hashes"`
	Options              []string     `json:"options"`
	ProbeASN             string       `json:"probe_asn"`
	ProbeCC              string       `json:"probe_cc"`
	ProbeCity            string       `json:"probe_city"`
	ProbeIP              string       `json:"probe_ip"`
	SoftwareName         string       `json:"software_name"`
	SoftwareVersion      string       `json:"software_version"`
	TestHelpers          interface{}  `json:"test_helpers"`
	TestKeys             interface{}  `json:"test_keys"`
	TestRuntime          float64      `json:"test_runtime"`
}

// Metadata contains metadata about the report
type Metadata struct {
	ReportID        string
	ProbeASN        string
	ProbeCC         string
	TestName        string
	SoftwareName    string
	SoftwareVersion string
	ReportFilePath  string
	CreationTime    time.Time
	LastUpdateTime  time.Time
	EntryCount      int64
	Closed          bool
}

const TimestampFormat = "20060102T150405Z"

// GenReportID generates a new report id
func GenReportID(asn string) string {
	return fmt.Sprintf("%s_%s_%s",
		time.Now().UTC().Format(TimestampFormat),
		asn,
		util.RandomStr(50),
	)
}
