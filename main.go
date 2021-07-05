package main

import (
	"flag"
	"log"
	"os"

	"github.com/poblish/boulevard/generation"
	"golang.org/x/tools/go/packages"
)

type packagesList []string

var packageFlags packagesList
var rulesOutputPath string
var dashboardOutputPath string
var sourcePath string

func main() {
	currentDir, err := os.Getwd()
	if err == nil {
		currentDir = "."
	}

	flag.Var(&packageFlags, "pkg", "Packages to scan")
	flag.StringVar(&sourcePath, "sourcePath", currentDir, "Source path")
	flag.StringVar(&rulesOutputPath, "rulesOutputPath", "alert_rules.yaml", "Rules output path")
	flag.StringVar(&dashboardOutputPath, "dashboardOutputPath", "grafana_dashboard.json", "Dashboard output path")
	flag.Parse()

	conf := packages.Config{
		Dir: sourcePath,
		Mode:  packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: false,
	}

	loadedPkgs, err := packages.Load(&conf, packageFlags...)
	if err != nil {
		log.Fatalf("Could not load packages %s", err)
	}

	generator := &generation.DashboardGenerator{}
	metrics, err := generator.DiscoverMetrics(loadedPkgs)
	if err != nil {
		log.Fatalf("Metrics discovery failed %s", err)
	}

	// FIXME Hardcoded name
	err = generator.GenerateAlertRules(rulesOutputPath)
	if err != nil {
		log.Fatalf("Alert rule generation failed %s", err)
	}

	// FIXME Hardcoded name
	err = generator.GenerateGrafanaDashboard(dashboardOutputPath, metrics)
	if err != nil {
		log.Fatalf("Generation failed %s", err)
	}
}

func (i *packagesList) String() string {
	return "my string representation"
}

func (i *packagesList) Set(value string) error {
	*i = append(*i, value)
	return nil
}
