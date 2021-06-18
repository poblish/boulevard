# Boulevard

[![Build Status](https://travis-ci.com/poblish/boulevard.svg?branch=master)](https://travis-ci.com/poblish/boulevard)
[![codecov](https://codecov.io/gh/poblish/boulevard/branch/master/graph/badge.svg?token=sgcOWUtDqa)](https://codecov.io/gh/poblish/boulevard)

Auto-generate Grafana dashboards and Prometheus alert rules via static analysis from usage of the [Promenade](https://github.com/poblish/promenade) Golang Prometheus client.

**Set up code:**

````golang
$ cd example
$ cat example.go

package main

import (
	"fmt"

	"github.com/poblish/promenade/api"
)

/*  Define two rules:

    @AlertDefaults(displayPrefix = Application, severity = warning, team = myTeam) => optional

    @ZeroToleranceErrorAlertRule(name = calcError, errorLabel="e", severity = pager, summary = Calculation error, description = "A calculation failed unexpectedly")

    @ElevatedErrorRateAlertRule(name = calcProblems, errorLabel="e", timeRange=10m, ratePerSecondThreshold=0.5, summary = More errors, description = "Too high error rate")
*/
func main() {
	metrics := api.NewMetrics(promenade.MetricOpts{MetricNamePrefix: "prefix"})
	metrics.Counter("c").Inc()
	metrics.CounterWithLabel("places", "city").IncLabel("London")
	metrics.CounterWithLabels("animals", []string{"type", "breed"}).IncLabel("cat", "persian")
	metrics.Error("e")
}
````

**Install:**

````bash
$ export PATH="$PATH:/Users/.../go/bin"
$ go get -v github.com/poblish/boulevard
````

**Generate dashboard:**

````bash
$ cd example
$ boulevard   ## optional --pkg github.com/my/pkg --rulesOutputPath rules/alert_rules.yaml --dashboardOutputPath dashboards/grafana_dashboard.json

{
  "annotations": {
    "list": [{
        "builtIn": 1, "datasource": "-- Grafana --", "enable": true, "hide": true, "iconColor": "rgba(0, 211, 255, 1)", "name": "Annotations & Alerts", "type": "dashboard"
      }]
  },
  "editable": true,
  "gnetId": null,
  "graphTooltip": 0,
  "id": 26,
  ...
````

**Generate validated alert rules YAML:**

````bash
$ boulevard

name: Application auto-generated alerts
rules:
- alert: ApplicationCalcError
  expr: sum(rate(prefix_errors{error_type='e'}[1m])) > 0
  duration: 1m
  labels:
    severity: pager
    team: myTeam
  annotations:
    description: A calculation failed unexpectedly
    summary: Calculation error
- alert: ApplicationCalcProblems
  expr: sum(rate(prefix_errors{error_type='e'}[10m])) > 0.5
  duration: 5m
  labels:
    severity: warning
    team: myTeam
  annotations:
    description: Too high error rate
    summary: More errors
````
