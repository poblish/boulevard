package generation

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	promenade "github.com/poblish/promenade/api"
	"github.com/stretchr/testify/assert"

	"golang.org/x/tools/go/packages"
)

var scanConf = packages.Config{
	Mode:  packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	Tests: true,
}

func TestBasic(t *testing.T) {
	loadedPkgs, err := packages.Load(&scanConf, "")
	assert.NoError(t, err)

	generator := &DashboardGenerator{}
	metrics, _ := generator.DiscoverMetrics(loadedPkgs)
	assert.Equal(t, len(metrics), 9)

	names := make([]string, len(metrics))
	for i, each := range metrics {
		names[i] = each.FullMetricName
	}

	assert.Equal(t, []string{"prefix_c", "prefix_places", "prefix_animals", "prefix_e", "prefix_g", "prefix_h", "prefix_hb", "prefix_s", "prefix_t"}, names)

	labels := make([]string, len(metrics))
	for i, each := range metrics {
		labels[i] = each.MetricLabels
	}

	assert.Equal(t, []string{"", " by (city)", " by (type,breed)", "", "", "", "", " by (quantile)", " by (quantile)"}, labels)

	panelTitles := make([]string, len(metrics))
	for i, each := range metrics {
		panelTitles[i] = each.PanelTitle
	}

	assert.Equal(t, []string{"c", "places", "animals", "e", "g", "h", "hb", "s", "t"}, panelTitles)
}

func TestCustomOptions(t *testing.T) {
	loadedPkgs, err := packages.Load(&scanConf, "github.com/poblish/boulevard/generation/test/d")
	assert.NoError(t, err)

	generator := &DashboardGenerator{}
	metrics, _ := generator.DiscoverMetrics(loadedPkgs)
	assert.Equal(t, len(metrics), 5)

	names := make([]string, len(metrics))
	for i, each := range metrics {
		names[i] = each.FullMetricName
	}

	assert.Equal(t, []string{"prefix:c", "prefix:places", "prefix:animals", "prefix:e", "prefix:g"}, names)

	labels := make([]string, len(metrics))
	for i, each := range metrics {
		labels[i] = each.MetricLabels
	}

	assert.Equal(t, []string{"", " by (city)", " by (type,breed)", "", ""}, labels)

	panelTitles := make([]string, len(metrics))
	for i, each := range metrics {
		panelTitles[i] = each.PanelTitle
	}

	assert.Equal(t, []string{"c", "places", "animals", "e", "g"}, panelTitles)
}

var expectedOutput = `
name: Application auto-generated alerts
rules:
- alert: ApplicationCalcError
  expr: sum(rate(prefix_errors{error_type='e'}[1m])) > 0
  duration: 10s
  labels:
    severity: pager
    team: myTeam
  annotations:
    description: A calculation failed unexpectedly
    summary: Calculation error
- alert: ApplicationCalcProblems
  expr: sum(rate(prefix_errors{error_type='e'}[10m])) > 1
  duration: 5m
  labels:
    severity: warning
    team: myTeam
  annotations:
    description: Too high error rate
    summary: More errors
`

func TestAlertRuleGeneration(t *testing.T) {
	loadedPkgs, err := packages.Load(&scanConf, "")
	assert.NoError(t, err)

	generator := &DashboardGenerator{}
	_, _ = generator.DiscoverMetrics(loadedPkgs)

	tempFile, err := ioutil.TempFile("", "x*.yaml")
	if err != nil {
		log.Fatal(err)
	}

	//goland:noinspection GoUnhandledErrorResult
	defer os.Remove(tempFile.Name())

	err = generator.GenerateAlertRules(tempFile.Name(), OutputOptions{AlertRuleFormat: PrometheusAlertManagerFormat})
	assert.NoError(t, err)
	assert.FileExists(t, tempFile.Name())

	bytes, _ := ioutil.ReadFile(tempFile.Name())
	assert.Equal(t, strings.TrimSpace(string(bytes)), strings.TrimSpace(expectedOutput))
}

func TestGrafanaDashboardGeneration(t *testing.T) {
	loadedPkgs, err := packages.Load(&scanConf, "")
	assert.NoError(t, err)

	generator := &DashboardGenerator{}
	metrics, _ := generator.DiscoverMetrics(loadedPkgs)

	tempFile, err := ioutil.TempFile("", "dash*.json")
	if err != nil {
		log.Fatal(err)
	}

	//goland:noinspection GoUnhandledErrorResult
	defer os.Remove(tempFile.Name())

	err = generator.GenerateGrafanaDashboard(tempFile.Name(), metrics)
	assert.NoError(t, err)
	assert.FileExists(t, tempFile.Name())

	bytes, _ := ioutil.ReadFile(tempFile.Name())
	data := strings.TrimSpace(string(bytes))
	// data, _ := json.Marshal(bytes)

	// FIXME Improve
	assert.Contains(t, data, `"targets": [{"expr": "sum(rate(prefix_places[15m])) by (city)", "intervalFactor": 1, "refId": "A"}],`)
	assert.Contains(t, data, `"targets": [{"expr": "sum(prefix_animals) by (type,breed)", "intervalFactor": 1, "refId": "A"}],`)
	assert.Contains(t, data, `"targets": [{"expr": "avg(prefix_t{quantile=~\"0.5|0.75|0.9|0.99\"}) by (quantile)", "format": "time_series", "intervalFactor": 1, "refId": "A"}],`)
}

func TestInvalidErrorLabelAnnotation(t *testing.T) {
	loadedPkgs, err := packages.Load(&scanConf, "github.com/poblish/boulevard/generation/test/a")
	assert.NoError(t, err)

	generator := &DashboardGenerator{}
	metrics, _ := generator.DiscoverMetrics(loadedPkgs)
	assert.Equal(t, len(metrics), 1)

	names := make([]string, len(metrics))
	for i, each := range metrics {
		names[i] = each.FullMetricName
	}

	assert.Equal(t, names, []string{"prefix_not_e"})

	tempFile, err := ioutil.TempFile("", "x*.yaml")
	if err != nil {
		log.Fatal(err)
	}

	//goland:noinspection GoUnhandledErrorResult
	defer os.Remove(tempFile.Name())

	err = generator.GenerateAlertRules(tempFile.Name(), OutputOptions{AlertRuleFormat: PrometheusAlertManagerFormat})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "alert refers to missing metric prefix_e")
}

func TestMultipleDefaultsAnnotations(t *testing.T) {
	loadedPkgs, err := packages.Load(&scanConf, "github.com/poblish/boulevard/generation/test/b")
	assert.NoError(t, err)

	generator := &DashboardGenerator{}
	_, err = generator.DiscoverMetrics(loadedPkgs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only one @AlertDefaults allowed per project")
}

func TestBadErrorRateAnnotations(t *testing.T) {
	loadedPkgs, err := packages.Load(&scanConf, "github.com/poblish/boulevard/generation/test/c")
	assert.NoError(t, err)

	generator := &DashboardGenerator{}
	_, _ = generator.DiscoverMetrics(loadedPkgs)

	tempFile, err := ioutil.TempFile("", "x*.yaml")
	if err != nil {
		log.Fatal(err)
	}
	//goland:noinspection ALL
	defer os.Remove(tempFile.Name())

	err = generator.GenerateAlertRules(tempFile.Name(), OutputOptions{AlertRuleFormat: PrometheusAlertManagerFormat})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad ratePerSecondThreshold: strconv.ParseFloat: parsing \"\": invalid syntax")
}

/*
	Metric names should be fully qualified (include prefix) if more than one Promenade metrics object / more than one prefix is in use
   	@AlertDefaults(displayPrefix = Application, severity = warning, team = myTeam)
	@ZeroToleranceErrorAlertRule(name = calcError, errorLabel="e", severity = pager, summary = Calculation error, description = "A calculation failed unexpectedly")
	@ElevatedErrorRateAlertRule(name = calcProblems, errorLabel="e", timeRange=10m, ratePerSecondThreshold=1, summary = More errors, description = "Too high error rate")
*/
//goland:noinspection GoUnusedFunction
func sampleMetricUsage() { // Is used!!
	metrics := promenade.NewMetrics(promenade.MetricOpts{MetricNamePrefix: "prefix"})
	metrics.Counter("c").Inc()
	metrics.Counter("c").Inc()
	metrics.CounterWithLabel("places", "city").IncLabel("London")
	metrics.CounterWithLabels("animals", []string{"type", "breed"}).IncLabel("cat", "persian")
	metrics.Error("e")
	metrics.Gauge("g")
	metrics.HistogramForResponseTime("h")
	metrics.Histogram("hb", []float64{1, 10})
	metrics.Summary("s").Observe(1212)
	metrics.Summary("s").Observe(233)
	timedMethod(&metrics)

	fmt.Println(metrics.TestHelper().MetricNames())
}

func timedMethod(metrics *promenade.PrometheusMetrics) { // Is used!!
	defer metrics.Timer("t")()
	fmt.Println("Whatever it is we're timing")
}
