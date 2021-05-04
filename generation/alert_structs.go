package generation

import (
	"log"
	"strconv"
)

type AlertDefaults struct {
	displayPrefix string
	team          string
	severity      string
}

type AlertRule interface {
	properties() map[string]string
	alertRuleExpression(metricPrefix string) string
}

type ZeroToleranceErrorAlertRule struct {
	AlertRule
	props map[string]string
}

func (r ZeroToleranceErrorAlertRule) properties() map[string]string {
	return r.props
}

func (r ZeroToleranceErrorAlertRule) alertRuleExpression(metricPrefix string) string {
	return "sum(rate(" + metricPrefix + "errors{error_type='" + r.props["errorLabel"] + "'}[" + r.props["timeRange"] + "])) > 0"
}

type ElevatedErrorRateAlertRule struct {
	AlertRule
	props map[string]string
}

func (r ElevatedErrorRateAlertRule) properties() map[string]string {
	return r.props
}

func (r ElevatedErrorRateAlertRule) alertRuleExpression(metricPrefix string) string {
	unvalidatedRate := r.props["ratePerSecondThreshold"]
	_, err := strconv.ParseFloat(unvalidatedRate, 64)
	if err != nil {
		log.Fatalf("Bad ratePerSecondThreshold: %v", err)
	}

	return "sum(rate(" + metricPrefix + "errors{error_type='" + r.props["errorLabel"] + "'}[" + r.props["timeRange"] + "])) > " + unvalidatedRate
}

// ====================================================================================

type AlertRulesGroup struct {
	Name  string
	Rules []AlertRuleOutput
}

type AlertRuleOutput struct {
	Alert       string
	Expr        string
	Duration    string
	Labels      map[string]string
	Annotations map[string]string
}
