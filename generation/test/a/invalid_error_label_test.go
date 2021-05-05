package a

import (
	promenade "github.com/poblish/promenade/api"
)

/*
	@ZeroToleranceErrorAlertRule(name = calcError, errorLabel="e", severity = pager, summary = Calculation error, description = "A calculation failed unexpectedly")
*/
func _() { // Is used!!
	metrics := promenade.NewMetrics(promenade.MetricOpts{MetricNamePrefix: "prefix"})
	metrics.Error("not_e")
}
