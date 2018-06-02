package handler

import "github.com/ooni/collector/collector/middleware"

var platformMetric = middleware.Metric{
	Name:        "platform_count",
	Description: "Counter of measurements per platform",
	Type:        "counter_vec",
	Args:        []string{"platform"},
}

var countryMetric = middleware.Metric{
	Name:        "country_count",
	Description: "Counter of measurements per country",
	Type:        "counter_vec",
	Args:        []string{"probe_cc"},
}

// CustomMetrics are ooni-collector specific metrics
var CustomMetrics = []*middleware.Metric{
	&platformMetric,
	&countryMetric,
}
