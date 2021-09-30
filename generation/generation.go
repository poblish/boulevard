package generation

import (
	"fmt"
	"go/ast"
	"log"
	"os"
	"path/filepath"
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

func (dg *DashboardGenerator) DiscoverMetrics(loadedPkgs []*packages.Package) ([]*metric, error) {
	nodeFilter := []ast.Node{(*ast.CommentGroup)(nil), (*ast.CompositeLit)(nil), (*ast.CallExpr)(nil)}

	var metrics []*metric

	dg.rawMetricPrefix = ""
	dg.currentMetricPrefix = ""
	dg.caseSensitiveMetricNames = false
	dg.alreadyGotErrors = false
	dg.foundMetricsObject = false
	dg.numPrefixesConfigured = 0

	var err error

	for _, eachPkg := range loadedPkgs {
		fmt.Println(">> Examining", eachPkg.PkgPath)

		inspector.New(eachPkg.Syntax).Preorder(nodeFilter, func(node ast.Node) {
			switch stmt := node.(type) {
			case *ast.CommentGroup:
				err = dg.processAlertAnnotations(stmt)

			case *ast.CompositeLit:
				// Discover... metricsOpts := promApi.MetricOpts{MetricNamePrefix: serviceName,} \n metrics := promApi.NewMetrics(metricsOpts)
				nodeType := eachPkg.TypesInfo.TypeOf(stmt)
				if nodeType.String() == "github.com/poblish/promenade/api.MetricOpts" {
					dg.discoverMetricOptions(eachPkg, stmt)
				}

			case *ast.CallExpr:

				if mthd, ok := stmt.Fun.(*ast.SelectorExpr); ok {

					if ident, ok := mthd.X.(*ast.Ident); ok {
						typeRef := eachPkg.TypesInfo.Uses[ident]

						// Don't do == on type in case of pointer prefix
						if typeRef != nil && strings.Contains(typeRef.Type().String(), PromenadePkg) && mthd.Sel.Name != "TestHelper" {
							metricName := stripQuotes(stmt.Args[0].(*ast.BasicLit).Value)

							newMetric := dg.interceptMetric(mthd.Sel.Name, metricName, stmt.Args)
							if newMetric != nil {
								metrics = append(metrics, newMetric)
							}

						} else {
							statementType := eachPkg.TypesInfo.Types[stmt].Type

							if mthd.Sel.Name == "NewMetrics" && statementType != nil && statementType.String() == PromenadePkg {
								// Parse the single argument to NewMetrics, deconstruct the Opts
								switch firstArg := stmt.Args[0].(type) {
								case *ast.CompositeLit:
									dg.discoverMetricOptions(eachPkg, firstArg)
								}
							}
						}
					} else if subExpr, ok := mthd.X.(*ast.SelectorExpr); ok /* Nested calls like `defer x.Timer()` */ {
						subExprTypeName := eachPkg.TypesInfo.Types[subExpr].Type

						if subExprTypeName != nil && strings.Contains(subExprTypeName.String(), PromenadePkg) {
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

	if err != nil {
		return metrics, err
	}

	if !dg.foundMetricsObject {
		log.Fatalf("ERROR: No Metrics found")
	}

	if len(metrics) < 1 {
		log.Printf("No Promenade metrics found")
		return metrics, nil
	}

	// Complete...
	dg.metricsIntercepted = make(map[string]bool)

	filteredIdx := 0

	for _, eachMetric := range metrics {
		eachMetric.MetricsPrefix = dg.currentMetricPrefix
		eachMetric.FullMetricName = dg.currentMetricPrefix + eachMetric.normalisedMetricName

		// Met this *full* name before?
		if _, ok := dg.metricsIntercepted[eachMetric.FullMetricName]; ok {
			continue
		}

		// Replace filtered item
		metrics[filteredIdx] = eachMetric
		filteredIdx++

		dg.metricsIntercepted[eachMetric.FullMetricName] = true

		// fmt.Println(eachMetric.metricCall, "=>", eachMetric.FullMetricName)
	}

	// Remove crud from the end of the slice
	for j := filteredIdx; j < len(metrics); j++ {
		metrics[j] = nil
	}
	metrics = metrics[:filteredIdx]

	fmt.Println(len(dg.metricsIntercepted), "metrics discovered")

	return metrics, nil
}

func (dg *DashboardGenerator) GenerateAlertRules(filePath string, options OutputOptions) error {
	return dg.RuleGenerator.postProcess(filePath, dg.currentMetricPrefix, dg.numPrefixesConfigured > 1, dg.rawMetricPrefix, dg.metricsIntercepted, options)
}

func (dg *DashboardGenerator) GenerateGrafanaDashboard(destFilePath string, metrics []*metric) error {
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

	if err := os.MkdirAll(filepath.Dir(destFilePath), os.ModePerm); err != nil {
		log.Fatalf("Output directory creation failed: %s", err)
	}

	outputFile, err := os.Create(destFilePath)
	if err != nil {
		log.Fatalf("Output file creation failed: %s", err)
	}

	fmt.Println("Writing dashboard to", FriendlyFileName(destFilePath))

	uid := truncateText(dg.currentMetricPrefix+"generated", 40)
	tErr := tmpl.Execute(outputFile, &dashboardData{Metrics: metrics, Title: dg.rawMetricPrefix + " Visualised Metrics", Id: uid})
	if tErr != nil {
		log.Fatalf("template execution: %s", tErr)
	}

	return outputFile.Close()
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

		if metricCall == "TimerWithLabel" {
			singleLabel := stripQuotes(metricCallArgs[1].(*ast.BasicLit).Value)
			metricLabelString = fmt.Sprintf(" by (%s,quantile)", singleLabel)
		} else {
			metricLabelString = " by (quantile)"
		}

	} else if strings.HasPrefix(metricCall, "Summary") {
		metricType = "summary"

		if metricCall == "SummaryWithLabels" {

			multipleLabels := metricCallArgs[1].(*ast.CompositeLit).Elts
			labelNames := make([]string, len(multipleLabels))

			for i, entry := range multipleLabels {
				labelNames[i] = stripQuotes(entry.(*ast.BasicLit).Value)
			}

			metricLabelString = fmt.Sprintf(" by (%s,quantile)", strings.Join(labelNames, ","))
		} else if metricCall == "SummaryWithLabel" {

			singleLabel := stripQuotes(metricCallArgs[1].(*ast.BasicLit).Value)
			metricLabelString = fmt.Sprintf(" by (%s,quantile)", singleLabel)
		} else {
			metricLabelString = " by (quantile)"
		}
	}

	return &metric{metricCall: metricCall, normalisedMetricName: normalisedMetricName, PanelTitle: metricName, MetricType: metricType, MetricLabels: metricLabelString}
}

func (dg *DashboardGenerator) discoverMetricOptions(pkg *packages.Package, stmt *ast.CompositeLit) {
	dg.foundMetricsObject = true

	rawPrefixSeparator := "_" // as per Prometheus lib standard

	for _, elt := range stmt.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {

			var literalValue string
			switch value := kv.Value.(type) {
			case *ast.BasicLit:
				literalValue = value.Value
			case *ast.Ident:
				// dereference the Ident...
				identValue := pkg.TypesInfo.Types[value].Value
				if identValue != nil {
					literalValue = identValue.String()
				} else {
					fmt.Printf("[WARNING] Could not resolve MetricOptions identifier [%s] - is the value a constant?\n", value)
				}
			}

			switch kv.Key.(*ast.Ident).Name {
			case "MetricNamePrefix":
				dg.rawMetricPrefix = stripQuotes(literalValue)
			case "PrefixSeparator":
				rawPrefixSeparator = stripQuotes(literalValue)
			case "CaseSensitiveMetricNames":
				dg.caseSensitiveMetricNames = true
			}
		}
	}

	dg.handleDiscoveredPrefix(rawPrefixSeparator)
}

func (dg *DashboardGenerator) handleDiscoveredPrefix(separator string) {
	newMetricPrefix := normaliseAndLowercaseName(dg.rawMetricPrefix)
	if newMetricPrefix != "" && !strings.HasSuffix(newMetricPrefix, separator) {
		newMetricPrefix += separator
	}

	if dg.currentMetricPrefix != newMetricPrefix {
		dg.currentMetricPrefix = newMetricPrefix
		dg.numPrefixesConfigured++
	}
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

func truncateText(s string, max int) string {
	if max > len(s) {
		return s
	}
	return s[:strings.LastIndexAny(s[:max], " .,:;-")]
}
