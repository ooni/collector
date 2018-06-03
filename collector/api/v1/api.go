package apiv1

import (
	"net/http"
	"strings"

	apexLog "github.com/apex/log"
	"github.com/gin-gonic/gin"
	"github.com/ooni/collector/collector/handler"
	"github.com/ooni/collector/collector/paths"
	"github.com/spf13/viper"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

var log = apexLog.WithFields(apexLog.Fields{
	"pkg": "apiv1",
	"cmd": "ooni-collector",
})

// BindAPI bind all the request handlers and middleware
func BindAPI(router *gin.Engine) error {
	p := ginprometheus.NewPrometheus("oonicollector", handler.CustomMetrics)
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
	router.POST("/report", handler.CreateReportHandler)
	router.PUT("/report", handler.DeprecatedUpdateReportHandler)
	router.POST("/report/:reportID", handler.UpdateReportHandler)
	router.POST("/report/:reportID/close", handler.CloseReportHandler)

	v1 := router.Group("/api/v1")
	v1.POST("/report", handler.CreateReportHandler)
	v1.POST("/report/:reportID", handler.UpdateReportHandler)
	v1.POST("/report/:reportID/close", handler.CloseReportHandler)
	v1.POST("/measurement", handler.SubmitMeasurementHandler)

	admin := router.Group("/admin", gin.BasicAuth(gin.Accounts{
		"admin": viper.GetString("api.admin-password"),
	}))
	admin.DELETE("/report-file/:filename", handler.DeleteReportFileHandler)
	admin.StaticFS("/report-files", http.Dir(paths.ReportDir()))
	return nil
}
