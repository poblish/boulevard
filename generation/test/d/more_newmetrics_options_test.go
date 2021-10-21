package d

import (
	promenade "github.com/poblish/promenade/api"
)

/*
   	@AlertDefaults(displayPrefix = ABC, severity = warning, team = myTeam)
	@ZeroToleranceErrorAlertRule(name = calcError, errorLabel="e", severity = pager, summary = Calculation error, description = "A calculation failed unexpectedly")
*/
//goland:noinspection GoUnusedFunction
func unused() { //nolint:unused,deadcode // Is used!!
	metrics := promenade.NewMetrics(promenade.MetricOpts{MetricNamePrefix: "prefix", PrefixSeparator: ":", CaseSensitiveMetricNames: false})
	metrics.Counter("c").Inc()
	metrics.CounterWithLabel("places", "city").IncLabel("London")
	metrics.CounterWithLabels("animals", []string{"type", "breed"}).IncLabel("cat", "persian")
	metrics.Error("e")
	metrics.Gauge("g")
}
