package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/ooni/collector/collector/paths"
)

// DeleteReportFileHandler is a handler for delete processed report files
func DeleteReportFileHandler(c *gin.Context) {
	reportFile := c.Param("filename")
	fullPath := filepath.Join(paths.ReportDir(), reportFile)
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
