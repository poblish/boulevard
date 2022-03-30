package generation

import (
	"fmt"
	"strconv"
)

type AlertDefaults struct {
	displayPrefix            string
	team                     string
	severity                 string
	runbookUrlAnnotationName string
}

type AlertRule interface {
	properties() map[string]string
	alertRuleExpression(metricPrefix string) (string, error)
}

type ZeroToleranceErrorAlertRule struct {
	AlertRule
	props map[string]string
}

func (r ZeroToleranceErrorAlertRule) properties() map[string]string {
	return r.props
}

func (r ZeroToleranceErrorAlertRule) alertRuleExpression(metricPrefix string) (string, error) {
	return "sum(rate(" + metricPrefix + "errors{error_type='" + r.props["errorLabel"] + "'}[" + r.props["timeRange"] + "])) > 0", nil
}

type ElevatedErrorRateAlertRule struct {
	AlertRule
	props map[string]string
}

func (r ElevatedErrorRateAlertRule) properties() map[string]string {
	return r.props
}

func (r ElevatedErrorRateAlertRule) alertRuleExpression(metricPrefix string) (string, error) {
	unvalidatedRate := r.props["ratePerSecondThreshold"]
	_, err := strconv.ParseFloat(unvalidatedRate, 64)
	if err != nil {
		return "", fmt.Errorf("bad ratePerSecondThreshold: %v", err)
	}

	return "sum(rate(" + metricPrefix + "errors{error_type='" + r.props["errorLabel"] + "'}[" + r.props["timeRange"] + "])) > " + unvalidatedRate, nil
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

// PrometheusOperatorRulesSpec https://github.com/prometheus-operator/prometheus-operator/blob/master/Documentation/api.md#prometheusrulespec
type PrometheusOperatorRulesSpec struct {
	Groups []PrometheusOperatorAlertRulesGroup
}

type PrometheusOperatorAlertRulesGroup struct {
	Name  string
	Rules []PrometheusOperatorAlertRuleOutput
}

type PrometheusOperatorAlertRuleOutput struct {
	Alert       string
	Expr        string
	For         string
	Labels      map[string]string
	Annotations map[string]string
}
