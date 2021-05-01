package analysis

import (
	"errors"
	"fmt"
	"go/ast"
	"log"
	"os"
	"strings"
	"text/template"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var DashboardGenerator = &analysis.Analyzer{
	Name:     "promenadeGrafanaDashboardGenerator",
	Doc:      "Generates Grafana Dashboards from Promenade (Prometheus) metrics via static",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const PromenadePkg = "github.com/poblish/promenade/api.PrometheusMetrics"

func run(pass *analysis.Pass) (interface{}, error) {
	fmt.Println(">>> Starting run", pass.Pkg)

	inspect, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("analyzer is not type *inspector.Inspector")
	}

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

	currentMetricPrefix := ""
	caseSensitiveMetricNames := false

	metrics := []metric{}

	dashboardColumnPosition := 0
	alreadyGotErrors := false

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		switch stmt := node.(type) {
		case *ast.CallExpr:
			if mthd, ok := stmt.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := mthd.X.(*ast.Ident); ok {
					typeName := pass.TypesInfo.Uses[ident].Type().String()

					// Don't do == on type in case of pointer prefix
					if strings.HasSuffix(typeName, PromenadePkg) && mthd.Sel.Name != "TestHelper" {
						metricCall := mthd.Sel.Name
						metricName := stripQuotes(stmt.Args[0].(*ast.BasicLit).Value)

						var fullMetricName string
						if caseSensitiveMetricNames {
							fullMetricName = currentMetricPrefix + normalizer.Replace(metricName)
						} else {
							fullMetricName = currentMetricPrefix + normaliseAndLowercaseName(metricName)

						}

						// fmt.Println(metricCall, metricName, "=>", fullMetricName)

						metricType := ""
						if strings.HasPrefix(metricCall, "Counter") {
							metricType = "counter"
						} else if strings.HasPrefix(metricCall, "Error") {
							if alreadyGotErrors {
								return
							}

							alreadyGotErrors = true
							metricType = "errors"

						} else if strings.HasPrefix(metricCall, "Gauge") {
							metricType = "gauge"
						} else if strings.HasPrefix(metricCall, "Histo") {
							metricType = "histogram"
						} else if strings.HasPrefix(metricCall, "Summary") || strings.HasPrefix(metricCall, "Timer") {
							metricType = "gauge"
						}

						metrics = append(metrics, metric{PanelColumn: dashboardColumnPosition, MetricsPrefix: currentMetricPrefix,
							FullMetricName: fullMetricName, PanelTitle: metricName, MetricType: metricType})

						dashboardColumnPosition = 12 - dashboardColumnPosition // of 24

					} else {
						statementType := pass.TypesInfo.Types[stmt].Type.String()
						if mthd.Sel.Name == "NewMetrics" && statementType == PromenadePkg {

							rawMetricPrefix := ""
							rawPrefixSeparator := "_" // as per Prometheus lib standard

							// Parse the single argument to NewMetrics, deconstruct the Opts
							for _, elt := range stmt.Args[0].(*ast.CompositeLit).Elts {
								if kv, ok := elt.(*ast.KeyValueExpr); ok {
									switch kv.Key.(*ast.Ident).Name {
									case "MetricNamePrefix":
										rawMetricPrefix = stripQuotes(kv.Value.(*ast.BasicLit).Value)
									case "PrefixSeparator":
										rawPrefixSeparator = stripQuotes(kv.Value.(*ast.BasicLit).Value)
									case "CaseSensitiveMetricNames":
										caseSensitiveMetricNames = true
									}
								}
							}

							currentMetricPrefix = normaliseAndLowercaseName(rawMetricPrefix)
							if currentMetricPrefix != "" && !strings.HasSuffix(currentMetricPrefix, rawPrefixSeparator) {
								currentMetricPrefix += rawPrefixSeparator
							}

							// fmt.Println("> NewMetrics", currentMetricPrefix)
						}
					}
				}
			}
		}
	})

	fmt.Println("<<< Outputting...", metrics)

	tmpl, _ := template.New("default").Parse(DefaultDashboardTemplate)
	// tmpl := template.Must(template.ParseGlob("/Users/andrewregan/Development/Go\\ work/promenade/templates/dashboard.json"))
	err := tmpl.Execute(os.Stdout, &dashboardData{Metrics: metrics, Title: "MyTitle", Id: "MyId"})
	if err != nil {
		log.Fatalf("template execution: %s", err)
	}

	return nil, nil
}

type dashboardData struct {
	Metrics []metric
	Title   string
	Id      string
}

type metric struct {
	PanelColumn    int
	MetricsPrefix  string
	MetricType     string
	FullMetricName string
	PanelTitle     string
}

var normalizer = strings.NewReplacer(".", "_", "-", "_", "#", "_", " ", "_")

func normaliseAndLowercaseName(name string) string {
	return strings.ToLower(normalizer.Replace(name))
}

func stripQuotes(s string) string {
	if len(s) > 0 && s[0] == '"' {
		s = s[1:]
	}
	if len(s) > 0 && s[len(s)-1] == '"' {
		s = s[:len(s)-1]
	}
	return strings.TrimSpace(s)
}
