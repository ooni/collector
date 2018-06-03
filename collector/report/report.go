package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/apex/log"
	"github.com/ooni/collector/collector/aws"
	"github.com/ooni/collector/collector/info"
	"github.com/ooni/collector/collector/paths"
	"github.com/ooni/collector/collector/storage"
	"github.com/ooni/collector/collector/util"
	"github.com/rs/xid"
)

// expiryTimers is a map of timers keyed to the ReportID. These are used to
// ensure that after a certain amount of time has elapsed reports are closed
var expiryTimers = make(map[string]*time.Timer)

// expiryTimeDuration is after how much time a report expires
var expiryTimeDuration = time.Duration(8) * time.Hour

// BackendExtra is serverside extra metadata
type BackendExtra struct {
	SubmissionTime time.Time `json:"submission_time"`
	MeasurementID  string    `json:"measurement_id"`
	ReportID       string    `json:"report_id"`
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

// closedReportPath is the final path of a report. The filename looks like this:
// 20180601T172750Z-ndt-20180601T172754Z_AS14080_iR5R39aBde9hAcE6kMw7rOCAF0iR63IPSGtcMWYj0QDHHujaXu-AS14080-CO-probe-0.2.0.json
func closedReportPath(meta *storage.ReportMetadata) string {
	return filepath.Join(paths.ReportDir(), fmt.Sprintf(
		"%s-%s-%s-%s-%s-probe-0.2.0.json",
		meta.CreationTime.Format(TimestampFormat),
		meta.TestName,
		meta.ReportID,
		meta.ProbeASN,
		meta.ProbeCC,
	))
}

// TimestampFormat is the string format for a timestamp, useful for generating
// report ids
const TimestampFormat = "20060102T150405Z"

// GenReportID generates a new report id
func GenReportID(asn string) string {
	return fmt.Sprintf("%s_%s_%s",
		time.Now().UTC().Format(TimestampFormat),
		asn,
		util.RandomStr(50),
	)
}

// CreateNewReport creates a new report
func CreateNewReport(store *storage.Storage, testName string, probeASN string, softwareName string, softwareVersion string) (string, error) {
	reportID := GenReportID(probeASN)
	tmpPath := filepath.Join(paths.TempReportDir(), reportID)
	meta := storage.ReportMetadata{
		ReportID:        reportID,
		TestName:        testName,
		ProbeASN:        probeASN,
		ProbeCC:         "",
		Platform:        "",
		SoftwareName:    softwareName,
		SoftwareVersion: softwareVersion,
		CreationTime:    time.Now().UTC(),
		LastUpdateTime:  time.Now().UTC(),
		ReportFilePath:  tmpPath,
		Closed:          false,
		EntryCount:      0,
	}
	store.SetReport(&meta)
	os.OpenFile(tmpPath, os.O_RDONLY|os.O_CREATE, 0700)

	expiryTimers[reportID] = time.AfterFunc(expiryTimeDuration, func() {
		CloseReport(store, reportID)
	})

	return meta.ReportID, nil
}

// CloseReport marks the report as closed and moves it into the final reports folder
func CloseReport(store *storage.Storage, reportID string) error {
	expiryTimers[reportID].Reset(expiryTimeDuration)

	meta, err := store.GetReport(reportID)
	if err != nil {
		return err
	}
	if meta.Closed == true {
		return ErrReportIsClosed
	}

	dstPath := closedReportPath(meta)
	if meta.EntryCount > 0 {
		err = os.Rename(meta.ReportFilePath, dstPath)
		if err != nil {
			return err
		}
	} else {
		// There is no need to keep closed empty reports
		os.Remove(meta.ReportFilePath)
	}
	meta.ReportFilePath = dstPath
	meta.Closed = true
	expiryTimers[reportID].Stop()

	if err = store.SetReport(meta); err != nil {
		return err
	}
	value, err := json.Marshal(meta)
	if err != nil {
		log.WithError(err).Error("failed to serialize meta")
		return nil
	}
	_, err = aws.SendMessage(aws.Session, string(value), "report")
	if err != nil {
		log.WithError(err).Error("failed to publish to aws SQS")
		return nil
	}

	return nil
}

func genMeasurementID() string {
	return xid.New().String()
}

func addBackendExtra(meta *storage.ReportMetadata, entry *MeasurementEntry) string {
	measurementID := genMeasurementID()
	entry.BackendVersion = info.Version
	entry.BackendExtra.SubmissionTime = meta.LastUpdateTime
	entry.BackendExtra.ReportID = meta.ReportID
	entry.BackendExtra.MeasurementID = measurementID
	return measurementID
}

var probeCCRegexp = regexp.MustCompile("^[A-Z]{2}$")

// WriteEntry will write an entry to report
func WriteEntry(store *storage.Storage, reportID string, entry *MeasurementEntry) (string, *storage.ReportMetadata, error) {
	expiryTimers[reportID].Reset(expiryTimeDuration)

	meta, err := store.GetReport(reportID)
	if err != nil {
		return "", nil, err
	}
	if meta.Closed == true {
		return "", nil, ErrReportIsClosed
	}
	if meta.ProbeCC == "" {
		if probeCCRegexp.MatchString(entry.ProbeCC) != true {
			return "", nil, errors.New("Invalid probe_cc")
		}
		meta.ProbeCC = entry.ProbeCC
	}
	if meta.Platform == "" && entry.Annotations != nil {
		annotations := entry.Annotations.(map[string]interface{})
		platform, ok := annotations["platform"].(string)
		if ok {
			meta.Platform = platform
		}
	}
	meta.LastUpdateTime = time.Now().UTC()
	meta.EntryCount++
	measurementID := addBackendExtra(meta, entry)

	f, err := os.OpenFile(meta.ReportFilePath, os.O_APPEND|os.O_WRONLY, 0700)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(entry)
	if err != nil {
		log.WithError(err).Error("Failed to encode measurement entry")
		return "", nil, err
	}

	if err = store.SetReport(meta); err != nil {
		return "", nil, err
	}

	return measurementID, meta, nil
}

// ReloadExpiryTimers is used to reload the timers for reports to expire
func ReloadExpiryTimers(store *storage.Storage) error {
	reportList, err := store.ListReports()
	if err != nil {
		log.WithError(err).Error("failed to list reports")
		return err
	}

	// We setup the timers so that pending reports will expire the
	// ExpiryTimeDuration after the server has been rebooted
	for _, meta := range reportList {
		expiryTimers[meta.ReportID] = time.AfterFunc(expiryTimeDuration, func() {
			CloseReport(store, meta.ReportID)
		})
	}
	return nil
}
