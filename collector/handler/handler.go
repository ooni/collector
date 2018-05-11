package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateReportHandler for report creation
func CreateReportHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"backend_version":     "XXX",
		"report_id":           "XXX",
		"test_helper_address": "XXX",
		"supported_formats":   "XXX",
	})
	return
}

func DeprecatedUpdateReportHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
	return
}

func UpdateReportHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
	return
}

func CloseReportHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
	return
}
