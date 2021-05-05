package generation

import (
	"fmt"
	"go/ast"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

type RuleGenerator struct {
	defaults   *AlertDefaults
	alertRules []AlertRule
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

func (rg *RuleGenerator) postProcess(filePath string, metricPrefix string, multiplePrefixesFound bool, defaultDisplayPrefix string, fqnsInUse map[string]bool) error {
	alertEntries := make([]AlertRuleOutput, len(rg.alertRules))

	var displayPrefix string
	if rg.defaults != nil && rg.defaults.displayPrefix != "" {
		displayPrefix = rg.defaults.displayPrefix
	} else {
		displayPrefix = strings.Title(defaultDisplayPrefix)
	}

	if displayPrefix == "" {
		displayPrefix = "Application"
	}

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
		annotations["summary"] = ruleProps["summary"] // FIXME use desc if blank

		expr, err := eachRule.alertRuleExpression(metricPrefix)
		if err != nil {
			return err
		}

		alertEntries[i] = AlertRuleOutput{Alert: alertName, Expr: expr, Duration: ruleProps["duration"], Labels: labels, Annotations: annotations}
	}

	alertRulesGroup := AlertRulesGroup{Name: displayPrefix + " auto-generated alerts", Rules: alertEntries}

	data, err := yaml.Marshal(&alertRulesGroup)
	if err != nil {
		return fmt.Errorf("alert marshalling error: %v", err)
	}

	err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("output error: %v", err)
	}

	return err
}

func (rg *RuleGenerator) parseZeroToleranceErrorAlertRule(comment string) {
	props := make(map[string]string)
	props["timeRange"] = "1m"
	props["duration"] = "1m"

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
