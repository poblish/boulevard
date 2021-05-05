package a

import (
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

func TestMultipleDefaultsAnnotations(t *testing.T) {
	loadedPkgs, err := packages.Load(&scanConf, "")
	assert.NoError(t, err)

	generator := &generation.DashboardGenerator{}
	_, err = generator.DiscoverMetrics(loadedPkgs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only one @AlertDefaults allowed per project")
}

/*
   	@AlertDefaults(displayPrefix = ABC, severity = warning, team = myTeam)
   	@AlertDefaults(displayPrefix = DEF, severity = warning, team = myTeam)
	@ZeroToleranceErrorAlertRule(name = calcError, errorLabel="e", severity = pager, summary = Calculation error, description = "A calculation failed unexpectedly")
*/
func invalidErrorLabelAnnotation() { // Is used!!
	metrics := promenade.NewMetrics(promenade.MetricOpts{MetricNamePrefix: "prefix"})
	metrics.Error("e")
}
