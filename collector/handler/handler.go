package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	apexLog "github.com/apex/log"
	"github.com/gin-gonic/gin"
	"github.com/ooni/collector/collector/paths"
	"github.com/ooni/collector/collector/report"
	"github.com/ooni/collector/storage"
	"github.com/rs/xid"
)

var log = apexLog.WithFields(apexLog.Fields{
	"pkg": "handler",
	"cmd": "ooni-collector",
})

const backendVersion = "2.0.0-alpha"

func createNewReport(store *storage.Storage, req CreateReportRequest) (string, error) {
	reportID := report.GenReportID(req.ProbeASN)
	tmpPath := filepath.Join(paths.TempReportDir(), reportID)
	meta := report.Metadata{
		ReportID:        reportID,
		TestName:        req.TestName,
		ProbeASN:        req.ProbeASN,
		ProbeCC:         "",
		SoftwareName:    req.SoftwareName,
		SoftwareVersion: req.SoftwareVersion,
		CreationTime:    time.Now().UTC(),
		LastUpdateTime:  time.Now().UTC(),
		ReportFilePath:  tmpPath,
		Closed:          false,
		EntryCount:      0,
	}
	store.SetReport(&meta)
	os.OpenFile(tmpPath, os.O_RDONLY|os.O_CREATE, 0700)

	report.ExpiryTimers[reportID] = time.AfterFunc(report.ExpiryTimeDuration, func() {
		CloseReport(store, reportID)
	})

	return meta.ReportID, nil
}

// CreateReportRequest what a client sends as a request to create a new report
type CreateReportRequest struct {
	SoftwareName    string `json:"software_name"`
	SoftwareVersion string `json:"software_version"`
	TestName        string `json:"test_name" binding:""`
	TestVersion     string `json:"test_version"`
	ProbeASN        string `json:"probe_asn"`
	Content         string `json:"content"`
}

var softwareVersionRegexp = regexp.MustCompile("^[0-9A-Za-z_.+-]+$")
var testNameRegexp = regexp.MustCompile("^[a-zA-Z0-9_\\- ]+$")
var probeASNRegexp = regexp.MustCompile("^AS[0-9]+$")
var probeCCRegexp = regexp.MustCompile("^[A-Z]{2}$")

func validateRequest(req *CreateReportRequest) error {
	if softwareVersionRegexp.MatchString(req.SoftwareName) != true {
		return errors.New("Invalid software_name")
	}
	if testNameRegexp.MatchString(req.TestName) != true {
		return errors.New("Invalid test_name")
	}
	if probeASNRegexp.MatchString(req.ProbeASN) != true {
		return errors.New("Invalid probe_asn")
	}
	return nil
}

// CreateReportHandler for report creation
func CreateReportHandler(c *gin.Context) {
	store := c.MustGet("Storage").(*storage.Storage)

	var req CreateReportRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reportID, err := createNewReport(store, req)
	if err != nil {
		// XXX check this against the spec
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backend_version":   backendVersion,
		"report_id":         reportID,
		"supported_formats": []string{"json"},
	})
	return
}

// DeprecatedUpdateReportHandler is for legacy clients
func DeprecatedUpdateReportHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
	return
}

// ErrReportIsClosed indicates the report has already been closed
var ErrReportIsClosed = errors.New("Report is already closed")

func genMeasurementID() string {
	return xid.New().String()
}

func addBackendExtra(meta *report.Metadata, entry *report.MeasurementEntry) string {
	measurementID := genMeasurementID()
	entry.BackendVersion = backendVersion
	entry.BackendExtra.SubmissionTime = meta.LastUpdateTime
	entry.BackendExtra.ReportID = meta.ReportID
	entry.BackendExtra.MeasurementID = measurementID
	return measurementID
}

func writeEntry(store *storage.Storage, reportID string, entry *report.MeasurementEntry) (string, error) {
	report.ExpiryTimers[reportID].Reset(report.ExpiryTimeDuration)

	meta, err := store.GetReport(reportID)
	if err != nil {
		return "", err
	}
	if meta.Closed == true {
		return "", ErrReportIsClosed
	}
	if meta.ProbeCC == "" {
		if probeCCRegexp.MatchString(entry.ProbeCC) != true {
			log.Debugf("entry: %v", entry)
			log.Debugf("Invalid probe cc: \"%s\"", entry.ProbeCC)
			return "", errors.New("Invalid probe_cc")
		}
		meta.ProbeCC = entry.ProbeCC
	}
	meta.LastUpdateTime = time.Now().UTC()
	meta.EntryCount++
	measurementID := addBackendExtra(meta, entry)

	f, err := os.OpenFile(meta.ReportFilePath, os.O_APPEND|os.O_WRONLY, 0700)
	if err != nil {
		return "", err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(entry)
	if err != nil {
		log.WithError(err).Error("Failed to encode measurement entry")
		return "", err
	}

	if err = store.SetReport(meta); err != nil {
		return "", err
	}

	return measurementID, nil
}

// UpdateReportRequest is used to update a report
type UpdateReportRequest struct {
	Content report.MeasurementEntry `json:"content" binding:"required"`
	Format  string                  `json:"format"`
}

// UpdateReportHandler appends to an open report
func UpdateReportHandler(c *gin.Context) {
	var err error

	store := c.MustGet("Storage").(*storage.Storage)
	reportID := c.Param("reportID")

	var req UpdateReportRequest
	if err = c.BindJSON(&req); err != nil {
		log.WithError(err).Error("failed to bindJSON")
		return
	}
	entry := req.Content

	measurementID, err := writeEntry(store, reportID, &entry)
	if err != nil {
		if err == storage.ErrReportNotFound {
			log.WithError(err).Debug("report not found error")
			// XXX use the correct return value
			c.JSON(http.StatusNotFound, gin.H{
				"status": "not found",
			})
		}
		log.WithError(err).Error("got an invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"measurement_id": measurementID,
	})
	return
}

// CloseReport marks the report as closed and moves it into the final reports folder
func CloseReport(store *storage.Storage, reportID string) error {
	report.ExpiryTimers[reportID].Reset(report.ExpiryTimeDuration)

	meta, err := store.GetReport(reportID)
	if err != nil {
		return err
	}
	if meta.Closed == true {
		return ErrReportIsClosed
	}

	dstPath := paths.ClosedReportPath(meta)
	if meta.EntryCount > 0 {
		err = os.Rename(meta.ReportFilePath, dstPath)
		if err != nil {
			return err
		}
	} else {
		// There is no need to keep closed empty reports
		os.Remove(meta.ReportFilePath)
	}
	meta.Closed = true
	report.ExpiryTimers[reportID].Stop()

	if err = store.SetReport(meta); err != nil {
		return err
	}

	return nil
}

// CloseReportHandler moves the report to the report-dir
func CloseReportHandler(c *gin.Context) {
	store := c.MustGet("Storage").(*storage.Storage)
	reportID := c.Param("reportID")

	err := CloseReport(store, reportID)
	if err != nil {
		// XXX return proper error
		c.JSON(http.StatusNotAcceptable, gin.H{
			"error": "something",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
	return
}

// SubmitMeasurementHandler is a handler for submitting a measurement in a
// single request
func SubmitMeasurementHandler(c *gin.Context) {
	store := c.MustGet("Storage").(*storage.Storage)
	var (
		entry    report.MeasurementEntry
		reportID string
	)

	shouldClose := c.DefaultQuery("close", "false") == "true"
	if err := c.BindJSON(&entry); err != nil {
		log.WithError(err).Error("failed to bindJSON")
		return
	}
	reportID = entry.ReportID
	createReq := CreateReportRequest{
		SoftwareName:    entry.SoftwareName,
		SoftwareVersion: entry.SoftwareVersion,
		TestName:        entry.TestName,
		TestVersion:     entry.TestVersion,
		ProbeASN:        entry.ProbeASN,
	}
	if err := validateRequest(&createReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}
	if reportID == "" {
		rid, err := createNewReport(store, createReq)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}
		reportID = rid
	}
	measurementID, err := writeEntry(store, reportID, &entry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}
	if shouldClose == true {
		CloseReport(store, reportID)
	}
	c.JSON(http.StatusOK, gin.H{
		"report_id":      reportID,
		"measurement_id": measurementID,
	})
}
