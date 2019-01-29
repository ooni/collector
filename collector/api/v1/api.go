package apiv1

import (
	"strings"

	apexLog "github.com/apex/log"
	"github.com/gin-gonic/gin"
	"github.com/ooni/collector/collector"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

var log = apexLog.WithFields(apexLog.Fields{
	"pkg": "apiv1",
	"cmd": "ooni-collector",
})

// BindAPI bind all the request handlers and middleware
func BindAPI(router *gin.Engine) error {
	p := ginprometheus.NewPrometheus("oonicollector", collector.CustomMetrics)
	ignoredParams := []string{"reportID", "filename"}
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		url := c.Request.URL.String()
		for _, param := range ignoredParams {
			for _, p := range c.Params {
				if p.Key == param {
					url = strings.Replace(url, p.Value, ":"+param, 1)
					break
				}
			}
		}
		return url
	}
	p.Use(router)

	// This is to support legacy clients
	router.POST("/report", collector.CreateReportHandler)
	router.PUT("/report", collector.DeprecatedUpdateReportHandler)
	router.POST("/report/:reportID", collector.UpdateReportHandler)
	router.POST("/report/:reportID/close", collector.CloseReportHandler)

	v1 := router.Group("/api/v1")
	v1.POST("/report", collector.CreateReportHandler)
	v1.POST("/report/:reportID", collector.UpdateReportHandler)
	v1.POST("/report/:reportID/close", collector.CloseReportHandler)
	v1.POST("/measurement", collector.SubmitMeasurementHandler)
	return nil
}
