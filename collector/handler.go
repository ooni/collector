package collector

import (
	"net/http"
	"regexp"

	"github.com/apex/log"
	"github.com/gin-gonic/gin"
)

// CreateReportReq is a client request for the POST /report API endpoint
type CreateReportReq struct {
	SoftwareName      string `json:"software_name"`
	SoftwareVersion   string `json:"software_version"`
	ProbeASN          string `json:"probe_asn"`
	ProbeCC           string `json:"probe_cc"`
	TestName          string `json:"test_name"`
	TestVersion       string `json:"test_version"`
	DataFormatVersion string `json:"data_format_version"`
	Format            string `json:"format"`
	// The below fields are optional
	TestStartTime string   `json:"test_start_time,omitempty"`
	InputHashes   []string `json:"input_hashes,omitempty"`
	TestHelper    string   `json:"test_helper,omitempty"`
	Content       string   `json:"content,omitempty"`
	ProbeIP       string   `json:"probe_ip,omitempty"`
}

var softwareNameRegexp = regexp.MustCompile("^[0-9A-Za-z_\\.+-]+$")
var testNameRegexp = regexp.MustCompile("^[a-zA-Z0-9_\\- ]+$")
var probeASNRegexp = regexp.MustCompile("^AS[0-9]{1,10}$")
var probeCCRegexp = regexp.MustCompile("^[A-Z]{2}$")

// CreateReportHandler for report creation
func CreateReportHandler(c *gin.Context) {
	store := c.MustGet("Storage").(*Storage)

	var req CreateReportReq

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Format == "" {
		req.Format = "json"
	}
	if req.Format != "json" && req.Format != "yaml" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format. Must be either json or yaml"})
		return
	}

	reportID, err := CreateNewReport(store, req.Format)
	if err != nil {
		// XXX check this against the spec
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backend_version":   Version,
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
	Content MeasurementEntry `json:"content" binding:"required"`
	Format  string           `json:"format"`
}

// UpdateReportHandler appends to an open report
func UpdateReportHandler(c *gin.Context) {
	var err error

	store := c.MustGet("Storage").(*Storage)
	reportID := c.Param("reportID")

	var req UpdateReportRequest
	if err = c.BindJSON(&req); err != nil {
		log.WithError(err).Error("failed to bindJSON")
		return
	}
	entry := req.Content

	measurementID, err := WriteEntry(store, reportID, &entry)
	if err != nil {
		if err == ErrReportNotFound {
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
	store := c.MustGet("Storage").(*Storage)
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
