package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/rs/xid"
)

// TimestampFormat is the string format for a timestamp, useful for generating
// report ids
const TimestampFormat = "20060102T150405Z"

// ActiveReport stores metadata about an active report. The metadat is immutable
// across the lifecycle of a report submission
type ActiveReport struct {
	ReportID        string    `json:"report_id"`
	CreationTime    time.Time `json:"creation_time"`
	LastUpdateTime  time.Time `json:"last_update_time"`
	ProbeASN        string    `json:"probe_asn"`
	ProbeCC         string    `json:"probe_cc"`
	Platform        string    `json:"platform"`
	TestName        string    `json:"test_name"`
	SoftwareName    string    `json:"software_name"`
	SoftwareVersion string    `json:"software_version"`

	expiryTimer *time.Timer
	mutex       sync.Mutex
}

// GenReportFilename creates a filename for the specified ActiveReport
func (a *ActiveReport) GenReportFilename() (string, error) {
	if probeCCRegexp.MatchString(a.ProbeCC) != true {
		return "", errors.New("Invalid probe_cc")
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s-probe-0.2.0.json",
		a.CreationTime.Format(TimestampFormat),
		a.TestName,
		a.ReportID,
		a.ProbeASN,
		a.ProbeCC), nil
}

var activeReports = make(map[string]*ActiveReport)

// expiryTimeDuration is after how much time a report expires
var expiryTimeDuration = time.Duration(8) * time.Hour

// BackendExtra is serverside extra metadata
type BackendExtra struct {
	SubmissionTime time.Time `json:"submission_time" bson:"submission_time"`
	MeasurementID  string    `json:"measurement_id" bson:"measurement_id"`
	ReportID       string    `json:"report_id" bson:"report_id"`
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
	MeasurementStartTime string       `json:"measurement_start_time,omitempty"`
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

// GenReportID generates a new report id
func GenReportID(asn string) string {
	return fmt.Sprintf("%s_%s_%s",
		time.Now().UTC().Format(TimestampFormat),
		asn,
		RandomStr(50),
	)
}

// CreateNewReport creates a new report
func CreateNewReport(store *Storage, testName string, probeASN string, softwareName string, softwareVersion string) (string, error) {
	reportID := GenReportID(probeASN)
	activeReport := ActiveReport{
		ReportID:        reportID,
		TestName:        testName,
		ProbeASN:        probeASN,
		ProbeCC:         "",
		Platform:        "",
		SoftwareName:    softwareName,
		SoftwareVersion: softwareVersion,
		CreationTime:    time.Now().UTC(),
		LastUpdateTime:  time.Now().UTC(),
	}
	// XXX put the report metadata into an activeReports in memory map
	err := store.CreateReportFile(reportID)
	if err != nil {
		log.WithError(err).Error("failed to allocate a ReportID")
		return "", err
	}
	activeReport.expiryTimer = time.AfterFunc(expiryTimeDuration, func() {
		CloseReport(store, reportID)
	})
	activeReports[reportID] = &activeReport
	return reportID, nil
}

// CloseReport marks the report as closed and moves it into the final reports folder
func CloseReport(store *Storage, reportID string) error {
	activeReport, ok := activeReports[reportID]
	if !ok {
		return ErrReportNotFound
	}
	activeReport.mutex.Lock()
	activeReport.expiryTimer.Stop()

	// We check this again to avoid a race
	if _, ok := activeReports[reportID]; !ok {
		return ErrReportNotFound
	}

	reportFilename, err := activeReport.GenReportFilename()
	if err != nil {
		log.WithError(err).Error("failed to generate filename")
		return err
	}

	err = store.CloseReportFile(reportID, reportFilename)
	if err != nil {
		log.WithError(err).Error("failed to close report")
		return err
	}

	delete(activeReports, reportID)
	activeReport.mutex.Unlock()

	return nil
}

func genMeasurementID() string {
	return xid.New().String()
}

func validateMetadata(meta *ActiveReport, entry *MeasurementEntry) error {
	return nil
}

func addBackendExtra(meta *ActiveReport, entry *MeasurementEntry) string {
	measurementID := genMeasurementID()
	entry.BackendVersion = Version
	entry.BackendExtra.SubmissionTime = meta.LastUpdateTime
	entry.BackendExtra.ReportID = meta.ReportID
	entry.BackendExtra.MeasurementID = measurementID
	return measurementID
}

// WriteEntry will write an entry to report
func WriteEntry(store *Storage, reportID string, entry *MeasurementEntry) (string, error) {
	var err error
	activeReport, ok := activeReports[reportID]
	if !ok {
		return "", ErrReportNotFound
	}
	activeReport.mutex.Lock()
	activeReport.expiryTimer.Reset(expiryTimeDuration)

	// We check this again to avoid a race
	if _, ok := activeReports[reportID]; !ok {
		return "", ErrReportNotFound
	}

	if activeReport.ProbeCC == "" {
		if probeCCRegexp.MatchString(entry.ProbeCC) != true {
			return "", errors.New("Invalid probe_cc")
		}
		activeReport.ProbeCC = entry.ProbeCC
	}
	if activeReport.Platform == "" && entry.Annotations != nil {
		annotations := entry.Annotations.(map[string]interface{})
		platform, ok := annotations["platform"].(string)
		if ok {
			activeReport.Platform = platform
		}
	}

	err = validateMetadata(activeReport, entry)
	if err != nil {
		log.WithError(err).Error("inconsistent metadata found")
		return "", err
	}
	measurementID := addBackendExtra(activeReport, entry)
	entryBytes, err := json.Marshal(entry)
	if err != nil {
		log.WithError(err).Error("could not serialize entry")
		return "", err
	}

	store.WriteToReportFile(reportID, entryBytes)
	if err != nil {
		log.WithError(err).Error("failed to insert into measurements table")
		return "", err
	}

	activeReport.LastUpdateTime = time.Now().UTC()
	activeReport.mutex.Unlock()

	return measurementID, nil
}
