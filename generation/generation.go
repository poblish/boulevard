package generation

import (
	"fmt"
	"go/ast"
	"log"
	"os"
	"strings"
	"text/template"

	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
)

const PromenadePkg = "github.com/poblish/promenade/api.PrometheusMetrics"

type DashboardGenerator struct {
	RuleGenerator
	rawMetricPrefix          string
	currentMetricPrefix      string
	caseSensitiveMetricNames bool
	alreadyGotErrors         bool
	foundMetricsObject       bool
	numPrefixesConfigured    int
	metricsIntercepted       map[string]bool
}

var globalIncrementingPanelId int

func (dg *DashboardGenerator) DiscoverMetrics(loadedPkgs []*packages.Package) []*metric {
	nodeFilter := []ast.Node{(*ast.CommentGroup)(nil), (*ast.CallExpr)(nil)}

	metrics := []*metric{}

	dg.rawMetricPrefix = ""
	dg.currentMetricPrefix = ""
	dg.caseSensitiveMetricNames = false
	dg.alreadyGotErrors = false
	dg.foundMetricsObject = false
	dg.numPrefixesConfigured = 0

	for _, eachPkg := range loadedPkgs {
		fmt.Println(">> Examining", eachPkg.PkgPath)

		inspector.New(eachPkg.Syntax).Preorder(nodeFilter, func(node ast.Node) {
			switch stmt := node.(type) {
			case *ast.CommentGroup:
				dg.processAlertAnnotations(stmt)
			case *ast.CallExpr:

				if mthd, ok := stmt.Fun.(*ast.SelectorExpr); ok {

					if ident, ok := mthd.X.(*ast.Ident); ok {
						typeName := eachPkg.TypesInfo.Uses[ident].Type().String()

						// Don't do == on type in case of pointer prefix
						if strings.HasSuffix(typeName, PromenadePkg) && mthd.Sel.Name != "TestHelper" {
							metricName := stripQuotes(stmt.Args[0].(*ast.BasicLit).Value)

							newMetric := dg.interceptMetric(mthd.Sel.Name, metricName, stmt.Args)
							if newMetric != nil {
								metrics = append(metrics, newMetric)
							}

						} else {
							statementType := eachPkg.TypesInfo.Types[stmt].Type.String()

							if mthd.Sel.Name == "NewMetrics" && statementType == PromenadePkg {

								dg.foundMetricsObject = true

								rawPrefixSeparator := "_" // as per Prometheus lib standard

								// Parse the single argument to NewMetrics, deconstruct the Opts
								for _, elt := range stmt.Args[0].(*ast.CompositeLit).Elts {
									if kv, ok := elt.(*ast.KeyValueExpr); ok {
										switch kv.Key.(*ast.Ident).Name {
										case "MetricNamePrefix":
											dg.rawMetricPrefix = stripQuotes(kv.Value.(*ast.BasicLit).Value)
										case "PrefixSeparator":
											rawPrefixSeparator = stripQuotes(kv.Value.(*ast.BasicLit).Value)
										case "CaseSensitiveMetricNames":
											dg.caseSensitiveMetricNames = true
										}
									}
								}

								dg.currentMetricPrefix = normaliseAndLowercaseName(dg.rawMetricPrefix)
								if dg.currentMetricPrefix != "" && !strings.HasSuffix(dg.currentMetricPrefix, rawPrefixSeparator) {
									dg.currentMetricPrefix += rawPrefixSeparator
								}

								dg.numPrefixesConfigured++
							}
						}
					} else if subExpr, ok := mthd.X.(*ast.SelectorExpr); ok /* Nested calls like `defer x.Timer()` */ {
						subExprTypeName := eachPkg.TypesInfo.Types[subExpr].Type.String()

						if strings.HasSuffix(subExprTypeName, PromenadePkg) && mthd.Sel.Name != "TestHelper" {
							metricName := stripQuotes(stmt.Args[0].(*ast.BasicLit).Value)

							newMetric := dg.interceptMetric(mthd.Sel.Name, metricName, stmt.Args)
							if newMetric != nil {
								metrics = append(metrics, newMetric)
							}
						}
					}
				}
			}
		})
	}

	if !dg.foundMetricsObject {
		log.Fatalf("ERROR: No Metrics found")
	}

	if len(metrics) < 1 {
		log.Fatalf("No Promenade metrics found")
	}

	// Complete...
	dg.metricsIntercepted = make(map[string]bool)

	for _, eachMetric := range metrics {
		eachMetric.MetricsPrefix = dg.currentMetricPrefix
		eachMetric.FullMetricName = dg.currentMetricPrefix + eachMetric.normalisedMetricName

		// Met this *full* name before?
		if _, ok := dg.metricsIntercepted[eachMetric.FullMetricName]; ok {
			continue
		}
		dg.metricsIntercepted[eachMetric.FullMetricName] = true

		fmt.Println(eachMetric.metricCall, "=>", eachMetric.FullMetricName)
	}

	return metrics
}

func (dg *DashboardGenerator) Generate(metrics []*metric) error {
	dg.RuleGenerator.postProcess(dg.currentMetricPrefix, dg.numPrefixesConfigured > 1, dg.rawMetricPrefix, dg.metricsIntercepted)

	// tmpl := template.Must(template.ParseGlob("/Users/andrewregan/Development/Go\\ work/promenade/templates/dashboard.json"))

	tmpl, _ := template.New("default").Funcs(template.FuncMap{

		"incrementingPanelId": func() int {
			globalIncrementingPanelId++
			return globalIncrementingPanelId
		},

		"panelColumn": func() int {
			return (globalIncrementingPanelId % 2) * 12 // Switch from left to right, 2 abreast
		},
	}).Parse(DefaultDashboardTemplate)

	tErr := tmpl.Execute(os.Stdout, &dashboardData{Metrics: metrics, Title: "MyTitle", Id: "MyId"})
	if tErr != nil {
		log.Fatalf("template execution: %s", tErr)
	}

	return tErr
}

func (dg *DashboardGenerator) interceptMetric(metricCall string, metricName string, metricCallArgs []ast.Expr) *metric {
	var normalisedMetricName string
	if dg.caseSensitiveMetricNames {
		normalisedMetricName = normalizer.Replace(metricName)
	} else {
		normalisedMetricName = normaliseAndLowercaseName(metricName)
	}

	metricType := ""
	metricLabelString := ""

	if strings.HasPrefix(metricCall, "Counter") {
		if metricCall == "CounterWithLabels" {

			multipleLabels := metricCallArgs[1].(*ast.CompositeLit).Elts
			labelNames := make([]string, len(multipleLabels))

			for i, entry := range multipleLabels {
				labelNames[i] = stripQuotes(entry.(*ast.BasicLit).Value)
			}

			metricLabelString = fmt.Sprintf(" by (%s)", strings.Join(labelNames, ","))

			metricType = "counter"
		} else if metricCall == "CounterWithLabel" {

			singleLabel := stripQuotes(metricCallArgs[1].(*ast.BasicLit).Value)
			metricLabelString = fmt.Sprintf(" by (%s)", singleLabel)

			metricType = "counter"
		} else {
			metricType = "counter"
		}

	} else if strings.HasPrefix(metricCall, "Error") {
		if dg.alreadyGotErrors {
			return nil
		}

		dg.alreadyGotErrors = true
		metricType = "errors"

	} else if strings.HasPrefix(metricCall, "Gauge") {
		metricType = "gauge"
	} else if strings.HasPrefix(metricCall, "Histo") {
		metricType = "histogram"
	} else if strings.HasPrefix(metricCall, "Timer") {
		metricType = "timer"
	} else if strings.HasPrefix(metricCall, "Summary") {
		metricType = "summary"
	}

	return &metric{metricCall: metricCall, normalisedMetricName: normalisedMetricName, PanelTitle: metricName, MetricType: metricType, MetricLabels: metricLabelString}
}

type dashboardData struct {
	Metrics []*metric
	Title   string
	Id      string
}

type metric struct {
	metricCall           string
	normalisedMetricName string

	MetricsPrefix  string
	MetricType     string
	MetricLabels   string
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