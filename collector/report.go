package collector

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/rs/xid"
)

// GetExpiryTimeDuration is after how much time a report expires
var GetExpiryTimeDuration = func() time.Duration {
	return time.Duration(8) * time.Hour
}

// TimestampFormat is the string format for a timestamp, useful for generating
// report ids
const TimestampFormat = "20060102T150405Z"

// GenReportID generates a new report id
func GenReportID() string {
	return fmt.Sprintf("%s_%s_%s",
		time.Now().UTC().Format(TimestampFormat),
		"AS00",
		RandomStr(50),
	)
}

// NewActiveReport creates an ActiveReport used to track reports that are
// currently in progress.
func NewActiveReport(format string) *ActiveReport {
	return &ActiveReport{
		ReportID:     GenReportID(),
		CreationTime: time.Now().UTC(),
		Format:       format,
		IsEmpty:      true,
	}
}

// ActiveReport stores metadata about an active report. The metadat is immutable
// across the lifecycle of a report submission
type ActiveReport struct {
	// These are required for generating the filename
	ReportID     string    `json:"report_id"`
	CreationTime time.Time `json:"creation_time"`
	ProbeASN     string    `json:"probe_asn"`
	ProbeCC      string    `json:"probe_cc"`
	TestName     string    `json:"test_name"`
	Format       string

	// These are for collecting metrics
	Platform        string `json:"platform"`
	SoftwareName    string `json:"software_name"`
	SoftwareVersion string `json:"software_version"`

	Path        string
	IsEmpty     bool
	expiryTimer *time.Timer
	mutex       sync.Mutex
}

func (a *ActiveReport) SetFromEntry(e *MeasurementEntry) error {
	a.ProbeCC = e.ProbeCC
	a.ProbeASN = e.ProbeASN
	a.TestName = e.TestName
	if e.Annotations != nil {
		annotations := e.Annotations.(map[string]interface{})
		platform, ok := annotations["platform"].(string)
		if ok {
			a.Platform = platform
		}
	}
	if err := a.Validate(); err != nil {
		return err
	}
	return nil
}

// Validate checks that we have a valid report submitted
func (a *ActiveReport) Validate() error {
	if testNameRegexp.MatchString(a.TestName) != true {
		return errors.New("invalid \"test_name\" field")
	}
	if probeASNRegexp.MatchString(a.ProbeASN) != true {
		return errors.New("invalid \"probe_asn\" field")
	}
	if probeCCRegexp.MatchString(a.ProbeCC) != true {
		return errors.New("invalid \"probe_cc\" field")
	}
	if stringInSlice(a.Format, supportedFormats) != true {
		return fmt.Errorf("invalid \"format\" field, %s", a.Format)
	}
	return nil
}

// IncomingFilename determines the filename of an incoming report
func (a *ActiveReport) IncomingFilename() string {
	if stringInSlice(a.Format, supportedFormats) != true {
		// Defensive coding
		panic("an unexpected value for format was found. Bailing...")
	}
	return fmt.Sprintf("%s.%s",
		a.ReportID,
		a.Format)
}

// SyncFilename creates a filename in the sync directory of the active report
func (a *ActiveReport) SyncFilename() (string, error) {
	// Defensive coding.
	if err := a.Validate(); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s-%s-%s-%s-probe-0.2.0.%s",
		a.CreationTime.Format(TimestampFormat),
		a.TestName,
		a.ReportID,
		a.ProbeASN,
		a.ProbeCC,
		a.Format), nil
}

// BackendExtra is serverside extra metadata
type BackendExtra struct {
	SubmissionTime time.Time `json:"submission_time" bson:"submission_time"`
	MeasurementID  string    `json:"measurement_id" bson:"measurement_id"`
	ReportID       string    `json:"report_id" bson:"report_id"`
}

// MeasurementEntry is the structure of measurements submitted by an OONI Probe client
type MeasurementEntry struct {
	// These values are added by the pipeline
	// BucketDate     string `json:"bucket_date"`
	// ReportFilename string `json:"report_filename"`

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

// CreateNewReport creates a new report
func CreateNewReport(store *Storage, format string) (string, error) {
	activeReport := NewActiveReport(format)
	// XXX put the report metadata into an activeReports in memory map
	err := store.CreateReportFile(activeReport)
	if err != nil {
		log.WithError(err).Error("failed to create a report file")
		return "", err
	}
	activeReport.expiryTimer = time.AfterFunc(GetExpiryTimeDuration(), func() {
		CloseReport(store, activeReport.ReportID)
	})
	store.activeReports[activeReport.ReportID] = activeReport
	return activeReport.ReportID, nil
}

// CloseReport marks the report as closed and moves it into the final reports folder
func CloseReport(store *Storage, reportID string) error {
	activeReport, ok := store.activeReports[reportID]
	if !ok {
		return ErrReportNotFound
	}
	activeReport.mutex.Lock()
	activeReport.expiryTimer.Stop()
	defer activeReport.mutex.Unlock()

	// We check this again to avoid a race
	if _, ok := store.activeReports[reportID]; !ok {
		return ErrReportNotFound
	}

	err := store.CloseReportFile(activeReport)
	if err != nil {
		log.WithError(err).Error("failed to close report")
		return err
	}

	delete(store.activeReports, reportID)

	return nil
}

func genMeasurementID() string {
	return xid.New().String()
}

func addBackendExtra(meta *ActiveReport, entry *MeasurementEntry) string {
	measurementID := genMeasurementID()
	entry.BackendVersion = Version
	entry.BackendExtra.SubmissionTime = meta.CreationTime
	entry.BackendExtra.ReportID = meta.ReportID
	entry.BackendExtra.MeasurementID = measurementID
	return measurementID
}

// WriteEntry will write an entry to report
func WriteEntry(store *Storage, reportID string, entry *MeasurementEntry) (string, error) {
	var err error
	activeReport, ok := store.activeReports[reportID]
	if !ok {
		return "", ErrReportNotFound
	}
	activeReport.mutex.Lock()
	activeReport.expiryTimer.Reset(GetExpiryTimeDuration())
	defer activeReport.mutex.Unlock()

	// We check this again to avoid a race
	if _, ok := store.activeReports[reportID]; !ok {
		return "", ErrReportNotFound
	}

	if activeReport.IsEmpty == true {
		err = activeReport.SetFromEntry(entry)
		if err != nil {
			log.WithError(err).Error("failed to set from entry")
			return "", err
		}
	}

	measurementID := addBackendExtra(activeReport, entry)
	entryBytes, err := json.Marshal(entry)
	if err != nil {
		log.WithError(err).Error("could not serialize entry")
		return "", err
	}
	entryBytes = append(entryBytes, []byte("\n")...)
	store.WriteToReportFile(activeReport, entryBytes)
	if err != nil {
		log.WithError(err).Error("failed to insert into measurements table")
		return "", err
	}

	return measurementID, nil
}

// LoadActiveReportFromFile will load an incoming temporary report file and make
// it into an ActiveReport
func LoadActiveReportFromFile(path string) (*ActiveReport, error) {
	filename := filepath.Base(path)
	p := strings.Split(filename, ".")
	if len(p) != 2 {
		return nil, fmt.Errorf("Inconsistent filename: \"%s\"", filename)
	}
	reportID, format := p[0], p[1]
	activeReport := ActiveReport{
		ReportID: reportID,
		Format:   format,
	}

	file, err := os.Open(path)
	if err != nil {
		log.WithError(err).Errorf("failed to open active report from file: %s", path)
		return nil, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		log.WithError(err).Error("failed to stat active report file.")
		return nil, err
	}
	activeReport.CreationTime = fi.ModTime().UTC()
	if fi.Size() == 0 {
		activeReport.IsEmpty = true
		return &activeReport, nil
	}
	reader := bufio.NewReader(file)

	line, err := reader.ReadString('\n')
	if err != nil {
		log.WithError(err).Error("failed to read first entry of report.")
		return nil, err
	}
	var entry MeasurementEntry
	json.Unmarshal([]byte(line), &entry)
	if err := activeReport.SetFromEntry(&entry); err != nil {
		log.WithError(err).Error("failed to set from entry.")
		return nil, err
	}
	return &activeReport, nil
}

// ReloadActiveReports will check the incoming dir to see if some incoming
// reports need to be reloading. This makes it possible to restart the server
// without loosing track of active reports.
func ReloadActiveReports(store *Storage) error {
	files, err := ioutil.ReadDir(store.IncomingDir())
	if err != nil {
		log.WithError(err).Error("failed to list incoming dir")
		return err
	}
	for _, file := range files {
		ar, err := LoadActiveReportFromFile(filepath.Join(store.IncomingDir(), file.Name()))
		if err != nil {
			log.WithError(err).Errorf("failed to load file: %s", file.Name())
			return err
		}
		ar.expiryTimer = time.AfterFunc(GetExpiryTimeDuration(), func() {
			CloseReport(store, ar.ReportID)
		})
		store.activeReports[ar.ReportID] = ar
	}
	return nil
}
