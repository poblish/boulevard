package a

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/poblish/boulevard/generation"
	promenade "github.com/poblish/promenade/api"
	"github.com/stretchr/testify/assert"

	"golang.org/x/tools/go/packages"
)

var scanConf = packages.Config{
	Mode:  packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	Tests: true,
}

func TestInvalidErrorLabelAnnotation(t *testing.T) {
	loadedPkgs, err := packages.Load(&scanConf, "")
	assert.NoError(t, err)

	generator := &generation.DashboardGenerator{}
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
	defer os.Remove(tempFile.Name())

	err = generator.GenerateAlertRules(tempFile.Name(), metrics)
	assert.Contains(t, err.Error(), "alert refers to missing metric prefix_e")
}

/*
	@ZeroToleranceErrorAlertRule(name = calcError, errorLabel="e", severity = pager, summary = Calculation error, description = "A calculation failed unexpectedly")
*/
func invalidErrorLabelAnnotation() { // Is used!!
	metrics := promenade.NewMetrics(promenade.MetricOpts{MetricNamePrefix: "prefix"})
	metrics.Error("not_e")
}
