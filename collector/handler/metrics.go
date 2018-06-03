package handler

import ginprometheus "github.com/zsais/go-gin-prometheus"

var platformMetric = ginprometheus.Metric{
	Name:        "platform_count",
	Description: "Counter of measurements per platform",
	Type:        "counter_vec",
	Args:        []string{"platform"},
}

var countryMetric = ginprometheus.Metric{
	Name:        "country_count",
	Description: "Counter of measurements per country",
	Type:        "counter_vec",
	Args:        []string{"probe_cc"},
}

// CustomMetrics are ooni-collector specific metrics
var CustomMetrics = []*ginprometheus.Metric{
	&platformMetric,
	&countryMetric,
}
