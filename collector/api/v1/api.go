package apiv1

import (
	apexLog "github.com/apex/log"
	"github.com/gin-gonic/gin"
	"github.com/ooni/collector/collector/handler"
)

var log = apexLog.WithFields(apexLog.Fields{
	"pkg": "apiv1",
	"cmd": "ooni-collector",
})

// BindAPI bind all the request handlers and middleware
func BindAPI(router *gin.Engine) error {

	// This is to support legacy clients
	router.POST("/report", handler.CreateReportHandler)
	router.PUT("/report", handler.DeprecatedUpdateReportHandler)
	router.POST("/report/:reportID", handler.UpdateReportHandler)
	router.POST("/report/:reportID/close", handler.CloseReportHandler)

	v1 := router.Group("/api/v1")

	v1.POST("/report", handler.CreateReportHandler)
	v1.POST("/report/:reportID", handler.UpdateReportHandler)
	v1.POST("/report/:reportID/close", handler.CloseReportHandler)

	return nil
}
