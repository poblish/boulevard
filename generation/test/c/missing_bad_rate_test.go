package c

import (
	promenade "github.com/poblish/promenade/api"
)

/*
	@ElevatedErrorRateAlertRule(name = calcProblems, errorLabel="e", timeRange=10m, MISSING RATE THRESHOLD=, summary = More errors, description = "Too high error rate")
*/
func _() { // Is used!!
	metrics := promenade.NewMetrics(promenade.MetricOpts{MetricNamePrefix: "prefix"})
	metrics.Error("e")
}
