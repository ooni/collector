package handler

import "github.com/ooni/collector/collector/middleware"

// PlatformMetric indicates how many measurements we see per platform
var PlatformMetric = middleware.Metric{
	Name:        "platform_count",
	Description: "Counter of measurements per platform",
	Type:        "counter_vec",
	Args:        []string{"platform"},
}
