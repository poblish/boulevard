# Boulevard

Auto-generate Grafana dashboards via static analysis from usage of the [Promenade](https://github.com/poblish/promenade) Golang Prometheus client.

**Set up code:**

````golang
$ cd example
$ cat example.go

package main

import (
	"fmt"

	"github.com/poblish/promenade/api"
)

func main() {
	metrics := api.NewMetrics(promenade.MetricOpts{MetricNamePrefix: "prefix"})
	metrics.Counter("c")
	metrics.CounterWithLabel("places", "city").IncLabel("London")
	metrics.CounterWithLabels("animals", []string{"type", "breed"}).IncLabel("cat", "persian")
	metrics.Error("e")
}
````

**Install:**

````bash
$ export PATH="$PATH:/Users/.../go/bin"
$ ./install.sh
````

**Generate:**

````bash
$ cd example
$ boulevard   ## optional --pkg github.com/my/pkg --pkg ...
$

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