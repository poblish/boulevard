package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/poblish/boulevard/generation"
	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v2"
)

type packagesList []string

var packageFlags packagesList
var rulesOutputPath string
var rulesOutputFormat string
var dashboardOutputPath string
var metricsLabelsPath string
var sourcePath string
var defaultMetricsPrefix string

var alertManagerOutputFormat = "alertManager"
var defaultRulesOutputFileName = "alert_rules.yaml"
var defaultGrafanaDashboardFileName = "grafana_dashboard.json"

func main() {
	currentDir, err := os.Getwd()
	if err == nil {
		currentDir = "."
	}

	state := BoulevardState{}

	// See if there's any upstream state to pick up
	stateFileBytes, err := ioutil.ReadFile(filepath.Join(sourcePath, ".boulevard_state"))
	if err == nil {
		if err := yaml.Unmarshal(stateFileBytes, &state); err != nil {
			log.Fatalf("Bad state %s", err)
		}
	}

	flag.Var(&packageFlags, "pkg", "Packages to scan")
	flag.StringVar(&sourcePath, "sourcePath", "", "Source path")
	flag.StringVar(&rulesOutputPath, "rulesOutputPath", "", "Rules output path")
	flag.StringVar(&rulesOutputFormat, "rulesOutputFormat", "", "Rules output format")
	flag.StringVar(&dashboardOutputPath, "dashboardOutputPath", "", "Dashboard output path")
	flag.StringVar(&metricsLabelsPath, "metricsLabelsPath", "", "Metrics labels path")
	flag.StringVar(&defaultMetricsPrefix, "defaultMetricsPrefix", "", "Metrics prefix fallback/default")
	flag.Parse()

	if rulesOutputPath == "" {
		if state.GeneratedChartDir != "" {
			rulesOutputPath = fmt.Sprintf("%s/includes/prometheus-rules/%s", state.GeneratedChartDir, defaultRulesOutputFileName)
		} else {
			rulesOutputPath = defaultRulesOutputFileName
		}
	}

	if dashboardOutputPath == "" {
		if state.GeneratedChartDir != "" {
			dashboardOutputPath = fmt.Sprintf("%s/includes/dashboards/%s", state.GeneratedChartDir, defaultGrafanaDashboardFileName)
		} else {
			dashboardOutputPath = defaultGrafanaDashboardFileName
		}
	}

	if sourcePath == "" {
		if state.SourcePath != "" {
			sourcePath = state.SourcePath
		} else {
			sourcePath = currentDir
		}
	}

	if len(packageFlags) == 0 && state.DefaultPkg != "" {
		packageFlags = []string{state.DefaultPkg}
	}

	if rulesOutputFormat == "" {
		if state.RulesOutputFormat != "" {
			rulesOutputFormat = state.RulesOutputFormat
		} else {
			rulesOutputPath = alertManagerOutputFormat
		}
	}

	if metricsLabelsPath == "" {
		if state.MetricsLabelsPath != "" {
			metricsLabelsPath = state.MetricsLabelsPath
		}
	}

	if defaultMetricsPrefix == "" {
		defaultMetricsPrefix = state.DefaultMetricsPrefix
	}

	var alertRuleFormat int
	switch rulesOutputFormat {
	case alertManagerOutputFormat:
		alertRuleFormat = generation.PrometheusAlertManagerFormat
	case "operator":
		alertRuleFormat = generation.PrometheusOperatorFormat
	default:
		log.Fatalf("Unsupported rules output format %s", rulesOutputFormat)
	}

	fmt.Printf("Examining packages %v from repository dir: %s\n", packageFlags, generation.FriendlyFileName(sourcePath))

	////////////////////////////////////////////

	conf := packages.Config{
		Dir:   sourcePath,
		Mode:  packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: false,
	}

	loadedPkgs, err := packages.Load(&conf, packageFlags...)
	if err != nil {
		log.Fatalf("Could not load packages %s", err)
	}

	generator := &generation.DashboardGenerator{DefaultMetricsPrefix: defaultMetricsPrefix}
	metrics, err := generator.DiscoverMetrics(loadedPkgs)
	if err != nil {
		log.Fatalf("Metrics discovery failed %s", err)
	}

	if len(metrics) < 1 {
		return // nothing to do
	}

	// FIXME Hardcoded name
	alertMetrics, err := generator.GenerateAlertRules(rulesOutputPath, generation.OutputOptions{AlertRuleFormat: alertRuleFormat})
	if err != nil {
		log.Fatalf("Alert rule generation failed %s", err)
	}

	// FIXME Hardcoded name
	err = generator.GenerateGrafanaDashboard(dashboardOutputPath, metrics, state.DashboardTags)
	if err != nil {
		log.Fatalf("Generation failed %s", err)
	}

	if metricsLabelsPath != "" {
		metricsOutput := generation.AlertMetricsOutput{AlertsCount: alertMetrics.Count, UniqueMetricsCount: len(metrics)}
		metricsOutput.WriteToFile(metricsLabelsPath)
	}
}

func (i *packagesList) String() string {
	return "my string representation"
}

func (i *packagesList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type BoulevardState struct {
	SourcePath           string
	GeneratedChartDir    string
	DefaultPkg           string
	DefaultMetricsPrefix string
	RulesOutputFormat    string
	MetricsLabelsPath    string
	DashboardTags        []string
}
