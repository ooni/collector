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
	v1 := router.Group("/api/v1")

	v1.POST("/report", handler.CreateReportHandler)
	v1.PUT("/report", handler.DeprecatedUpdateReportHandler)
	v1.POST("/report/:report_id", handler.UpdateReportHandler)
	v1.POST("/report/:report_id/close", handler.CloseReportHandler)

	return nil
}
