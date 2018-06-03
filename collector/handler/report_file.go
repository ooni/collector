package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/ooni/collector/collector/paths"
)

var filenameRegexp = regexp.MustCompile("^[0-9A-Za-z_\\.+-]+$")

// DeleteReportFileHandler is a handler for delete processed report files
func DeleteReportFileHandler(c *gin.Context) {
	filename := c.Param("filename")
	if filenameRegexp.MatchString(filename) != true {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid filename",
		})
		return
	}

	fullPath := filepath.Join(paths.ReportDir(), filename)
	err := os.Remove(fullPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "deleted",
	})
	return
}
