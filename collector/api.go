package collector

import (
	"strings"

	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

// BindAPI bind all the request handlers and middleware
func BindAPI(router *gin.Engine) error {
	p := ginprometheus.NewPrometheus("oonicollector", CustomMetrics)
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
	router.POST("/report", CreateReportHandler)
	router.PUT("/report", DeprecatedUpdateReportHandler)
	router.POST("/report/:reportID", UpdateReportHandler)
	router.POST("/report/:reportID/close", CloseReportHandler)

	v1 := router.Group("/api/v1")
	v1.POST("/report", CreateReportHandler)
	v1.POST("/report/:reportID", UpdateReportHandler)
	v1.POST("/report/:reportID/close", CloseReportHandler)
	v1.POST("/measurement", SubmitMeasurementHandler)
	return nil
}
