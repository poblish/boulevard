package generation

import (
	"fmt"
	"go/ast"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type RuleGenerator struct {
	defaults   *AlertDefaults
	alertRules []AlertRule
}

const (
	PrometheusAlertManagerFormat = iota
	PrometheusOperatorFormat
)

type OutputOptions struct {
	AlertRuleFormat int
}

func (rg *RuleGenerator) processAlertAnnotations(commentGroup *ast.CommentGroup) error {
	if commentGroup != nil {
		for _, comment := range commentGroup.List {
			for _, eachLine := range strings.Split(strings.ReplaceAll(comment.Text, "\r\n", "\n"), "\n") {
				if strings.Contains(eachLine, "@ZeroToleranceErrorAlertRule") {
					rg.parseZeroToleranceErrorAlertRule(eachLine)
				} else if strings.Contains(eachLine, "@ElevatedErrorRateAlertRule") {
					rg.parseElevatedErrorRateAlertRule(eachLine)
				} else if strings.Contains(eachLine, "@AlertDefaults") {
					if rg.defaults != nil {
						return fmt.Errorf("only one @AlertDefaults allowed per project") // surely too strict...
					}
					rg.parseAlertDefaults(eachLine)
				}
			}
		}
	}

	return nil
}

var prefixNormalizer = strings.NewReplacer("_", "", "-", "", " ", "")

func (rg *RuleGenerator) postProcess(destFilePath string, metricPrefix string, multiplePrefixesFound bool, defaultDisplayPrefix string, fqnsInUse map[string]bool, options OutputOptions) error {

	var alertEntries []AlertRuleOutput
	var operatorAlertEntries []PrometheusOperatorAlertRuleOutput

	switch options.AlertRuleFormat {
	case PrometheusAlertManagerFormat:
		alertEntries = make([]AlertRuleOutput, len(rg.alertRules))
	case PrometheusOperatorFormat:
		operatorAlertEntries = make([]PrometheusOperatorAlertRuleOutput, len(rg.alertRules))
	}

	var displayPrefix string
	if rg.defaults != nil && rg.defaults.displayPrefix != "" {
		displayPrefix = rg.defaults.displayPrefix
	} else {
		displayPrefix = strings.Title(defaultDisplayPrefix)
	}

	if displayPrefix == "" {
		displayPrefix = "Application"
	}

	displayPrefix = prefixNormalizer.Replace(displayPrefix)

	for i, eachRule := range rg.alertRules {

		ruleProps := eachRule.properties()

		if rg.defaults != nil {
			if _, ok := ruleProps["team"]; !ok {
				ruleProps["team"] = rg.defaults.team
			}

			if _, ok := ruleProps["severity"]; !ok {
				ruleProps["severity"] = rg.defaults.severity
			}
		}

		var normalisedMetricName string
		if false { // FIXME dg.caseSensitiveMetricNames {
			normalisedMetricName = normalizer.Replace(ruleProps["errorLabel"])
		} else {
			normalisedMetricName = normaliseAndLowercaseName(ruleProps["errorLabel"])
		}

		var alertMetricFqn string
		if multiplePrefixesFound {
			alertMetricFqn = normalisedMetricName
		} else {
			alertMetricFqn = metricPrefix + normalisedMetricName
		}

		// Validate errorLabel is an actual metric name
		if _, ok := fqnsInUse[alertMetricFqn]; !ok {
			return fmt.Errorf("alert refers to missing metric %s", alertMetricFqn)
		}

		alertName := displayPrefix + strings.Title(ruleProps["name"])

		labels := make(map[string]string)
		labels["severity"] = ruleProps["severity"] // FIXME check blank
		labels["team"] = ruleProps["team"]         // FIXME check blank

		annotations := make(map[string]string)
		annotations["description"] = ruleProps["description"]

		// Use desc as summary if not otherwise set
		if ruleProps["summary"] != "" {
			annotations["summary"] = ruleProps["summary"]
		} else if ruleProps["description"] != "" {
			annotations["summary"] = ruleProps["description"]
		} else {
			return fmt.Errorf("no summary or description for alert %s", alertName)
		}

		expr, err := eachRule.alertRuleExpression(metricPrefix)
		if err != nil {
			return err
		}

		switch options.AlertRuleFormat {
		case PrometheusAlertManagerFormat:
			alertEntries[i] = AlertRuleOutput{Alert: alertName, Expr: expr, Duration: ruleProps["duration"], Labels: labels, Annotations: annotations}
		case PrometheusOperatorFormat:
			operatorAlertEntries[i] = PrometheusOperatorAlertRuleOutput{Alert: alertName, Expr: expr, For: ruleProps["duration"], Labels: labels, Annotations: annotations}
		}
	}

	var alertRulesSpec interface{}

	switch options.AlertRuleFormat {
	case PrometheusAlertManagerFormat:
		alertRulesSpec = AlertRulesGroup{Name: displayPrefix + " auto-generated alerts", Rules: alertEntries}
	case PrometheusOperatorFormat:
		alertRulesSpec = PrometheusOperatorRulesSpec{Groups: []PrometheusOperatorAlertRulesGroup{{Name: displayPrefix + " auto-generated alerts", Rules: operatorAlertEntries}}}
	}

	data, err := yaml.Marshal(&alertRulesSpec)
	if err != nil {
		return fmt.Errorf("alert marshalling error: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(destFilePath), os.ModePerm); err != nil {
		log.Fatalf("Output directory creation failed: %s", err)
	}

	fmt.Println("Writing alert rules to", FriendlyFileName(destFilePath))

	err = ioutil.WriteFile(destFilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("output error: %v", err)
	}

	return err
}

func (rg *RuleGenerator) parseZeroToleranceErrorAlertRule(comment string) {
	props := make(map[string]string)
	props["timeRange"] = "1m"
	props["duration"] = "10s" // FIXME make configurable

	parsePayload(comment, props)

	rg.alertRules = append(rg.alertRules, ZeroToleranceErrorAlertRule{props: props})
}

func (rg *RuleGenerator) parseElevatedErrorRateAlertRule(comment string) {
	props := make(map[string]string)
	props["timeRange"] = "5m"
	props["duration"] = "5m"

	parsePayload(comment, props)

	rg.alertRules = append(rg.alertRules, ElevatedErrorRateAlertRule{props: props})
}

func (rg *RuleGenerator) parseAlertDefaults(comment string) {
	props := make(map[string]string)
	parsePayload(comment, props)

	rg.defaults = &AlertDefaults{displayPrefix: props["displayPrefix"], team: props["team"], severity: props["severity"]}
}

func extractPayload(comment string) string {
	return comment[strings.Index(comment, "(")+1 : strings.Index(comment, ")")]
}

func parsePayload(payload string, props map[string]string) {
	for _, val := range strings.Split(extractPayload(payload), ",") {
		parts := strings.Split(val, "=")
		props[strings.TrimSpace(parts[0])] = stripQuotes(strings.TrimSpace(parts[1]))
	}
}
