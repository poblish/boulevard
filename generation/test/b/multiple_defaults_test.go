package b

import (
	promenade "github.com/poblish/promenade/api"
)

/*
   	@AlertDefaults(displayPrefix = ABC, severity = warning, team = myTeam)
   	@AlertDefaults(displayPrefix = DEF, severity = warning, team = myTeam)
	@ZeroToleranceErrorAlertRule(name = calcError, errorLabel="e", severity = pager, summary = Calculation error, description = "A calculation failed unexpectedly")
*/
func _() { // Is used!!
	metrics := promenade.NewMetrics(promenade.MetricOpts{MetricNamePrefix: "prefix"})
	metrics.Error("e")
}
