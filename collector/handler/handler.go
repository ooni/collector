package handler

import (
	"errors"
	"net/http"
	"regexp"

	apexLog "github.com/apex/log"
	"github.com/gin-gonic/gin"
	"github.com/ooni/collector/collector/info"
	"github.com/ooni/collector/collector/report"
	"github.com/ooni/collector/collector/storage"
)

var log = apexLog.WithFields(apexLog.Fields{
	"pkg": "handler",
	"cmd": "ooni-collector",
})

// CreateReportRequest what a client sends as a request to create a new report
type CreateReportRequest struct {
	SoftwareName    string `json:"software_name"`
	SoftwareVersion string `json:"software_version"`
	TestName        string `json:"test_name" binding:""`
	TestVersion     string `json:"test_version"`
	ProbeASN        string `json:"probe_asn"`
	Content         string `json:"content"`
}

var softwareNameRegexp = regexp.MustCompile("^[0-9A-Za-z_\\.+-]+$")
var testNameRegexp = regexp.MustCompile("^[a-zA-Z0-9_\\- ]+$")
var probeASNRegexp = regexp.MustCompile("^AS[0-9]+$")

func validateRequest(req *CreateReportRequest) error {
	if softwareNameRegexp.MatchString(req.SoftwareName) != true {
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

	reportID, err := report.CreateNewReport(store, req.TestName, req.ProbeASN, req.SoftwareName, req.SoftwareVersion)
	if err != nil {
		// XXX check this against the spec
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backend_version":   info.Version,
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

	measurementID, err := report.WriteEntry(store, reportID, &entry)
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
	// XXX temporarily disabled
	//platformMetric.MetricCollector.(*prometheus.CounterVec).WithLabelValues(meta.Platform).Inc()
	//countryMetric.MetricCollector.(*prometheus.CounterVec).WithLabelValues(meta.ProbeCC).Inc()

	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"measurement_id": measurementID,
	})
	return
}

// CloseReportHandler moves the report to the report-dir
func CloseReportHandler(c *gin.Context) {
	store := c.MustGet("Storage").(*storage.Storage)
	reportID := c.Param("reportID")

	err := report.CloseReport(store, reportID)
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
		rid, err := report.CreateNewReport(store, createReq.TestName,
			createReq.ProbeASN, createReq.SoftwareName, createReq.SoftwareVersion)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}
		reportID = rid
	}
	measurementID, err := report.WriteEntry(store, reportID, &entry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}
	if shouldClose == true {
		report.CloseReport(store, reportID)
	}
	c.JSON(http.StatusOK, gin.H{
		"report_id":      reportID,
		"measurement_id": measurementID,
	})
}
