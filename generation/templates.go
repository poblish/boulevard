package generation

const DefaultDashboardTemplate = `{{define "counter_gauge_cumulative"}}
{
  "bars": false,
  "dashLength": 10,
  "dashes": false,
  "datasource": "Prometheus",
  "fill": 1,
  "gridPos": {"h": 9,"w": 12,"x": {{ panelColumn }},"y": 0},
  "id": {{ incrementingPanelId }},
  "legend": {"avg": false,"current": false,"max": false,"min": false,"show": true,"total": false,"values": false},
  "lines": true,
  "linewidth": 1,
  "percentage": false,
  "pointradius": 5,
  "points": false,
  "seriesOverrides": [],
  "spaceLength": 10,
  "stack": false,
  "targets": [{"expr": "sum({{ .FullMetricName }}){{ .MetricLabels }}", "intervalFactor": 1, "refId": "A"}],
  "thresholds": [],
  "timeFrom": null,
  "timeRegions": [],
  "timeShift": null,
  "title": "{{ .PanelTitle }} (cumulative)",
  "tooltip": {"shared": true,"sort": 0,"value_type": "individual"},
  "type": "graph",
  "xaxis": {"buckets": null,"mode": "time","name": null,"show": true,"values": []},
  "yaxes": [{"format": "short", "label": null, "logBase": 1, "max": null, "min": null, "show": true},{"format": "short", "label": null, "logBase": 1, "max": null, "min": null, "show": true}],
  "yaxis": {"align": false,"alignLevel": null}
}
{{end}}

{{define "counter_gauge_rate"}}
{
  "bars": false,
  "dashLength": 10,
  "dashes": false,
  "datasource": "Prometheus",
  "fill": 1,
  "gridPos": {"h": 9,"w": 12,"x": {{ panelColumn }},"y": 0},
  "id": {{ incrementingPanelId }},
  "legend": {"avg": false,"current": false,"max": false,"min": false,"show": true,"total": false,"values": false},
  "lines": true,
  "linewidth": 1,
  "percentage": false,
  "pointradius": 5,
  "points": false,
  "seriesOverrides": [],
  "spaceLength": 10,
  "stack": false,
  "targets": [{"expr": "sum(rate({{ .FullMetricName }}[15m])){{ .MetricLabels }}", "intervalFactor": 1, "refId": "A"}],
  "thresholds": [],
  "timeFrom": null,
  "timeRegions": [],
  "timeShift": null,
  "title": "{{ .PanelTitle }} (rate)",
  "tooltip": {"shared": true,"sort": 0,"value_type": "individual"},
  "type": "graph",
  "xaxis": {"buckets": null,"mode": "time","name": null,"show": true,"values": []},
  "yaxes": [{"format": "short", "label": null, "logBase": 1, "max": null, "min": null, "show": true},{"format": "short", "label": null, "logBase": 1, "max": null, "min": null, "show": true}],
  "yaxis": {"align": false,"alignLevel": null}
}
{{end}}

{{define "errors"}}
{
  "bars": false,
  "dashLength": 10,
  "dashes": false,
  "datasource": "Prometheus",
  "fill": 1,
  "gridPos": {"h": 9,"w": 12,"x": {{ panelColumn }},"y": 0},
  "id": {{ incrementingPanelId }},
  "legend": {"avg": false,"current": false,"max": false,"min": false,"show": true,"total": false,"values": false},
  "lines": true,
  "linewidth": 1,
  "percentage": false,
  "pointradius": 5,
  "points": false,
  "seriesOverrides": [],
  "spaceLength": 10,
  "stack": false,
  "targets": [{"expr": "sum({{ .MetricsPrefix }}errors) by (error_type)", "intervalFactor": 1, "refId": "A"}],
  "thresholds": [],
  "timeFrom": null,
  "timeRegions": [],
  "timeShift": null,
  "title": "Errors by type",
  "tooltip": {"shared": true,"sort": 0,"value_type": "individual"},
  "type": "graph",
  "xaxis": {"buckets": null,"mode": "time","name": null,"show": true,"values": []},
  "yaxes": [{"format": "short", "label": null, "logBase": 1, "max": null, "min": null, "show": true},{"format": "short", "label": null, "logBase": 1, "max": null, "min": null, "show": true}],
  "yaxis": {"align": false,"alignLevel": null}
}
{{end}}

{{define "summary_timer"}}
{
  "bars": false,
  "dashLength": 10,
  "dashes": false,
  "datasource": "Prometheus",
  "fill": 1,
  "gridPos": {"h": 9,"w": 12,"x": {{ panelColumn }},"y": 0},
  "id": {{ incrementingPanelId }},
  "legend": {"avg": false,"current": false,"max": false,"min": false,"show": true,"total": false,"values": false},
  "lines": true,
  "linewidth": 1,
  "percentage": false,
  "pointradius": 5,
  "points": false,
  "seriesOverrides": [],
  "spaceLength": 10,
  "stack": false,
  "targets": [{"expr": "avg({{ .FullMetricName }}{ {{- .ExtraLabelFilter }}quantile=~\"0.5|0.75|0.9|0.99\"}){{ .MetricLabels }}", "format": "time_series", "intervalFactor": 1, "refId": "A"}],
  "thresholds": [],
  "timeFrom": null,
  "timeRegions": [],
  "timeShift": null,
  "title": "{{ .PanelTitle }}",
  "tooltip": {"shared": true,"sort": 0,"value_type": "individual"},
  "type": "graph",
  "xaxis": {"buckets": null,"mode": "time","name": null,"show": true,"values": []},
  "yaxes": [{"format": "dtdurations", "label": null, "logBase": 1, "max": null, "min": null, "show": true},{"format": "dtdurations", "label": null, "logBase": 1, "max": null, "min": null, "show": true}],
  "yaxis": {"align": false,"alignLevel": null}
}
{{end}}

{
  "annotations": {
    "list": [{
        "builtIn": 1, "datasource": "-- Grafana --", "enable": true, "hide": true, "iconColor": "rgba(0, 211, 255, 1)", "name": "Annotations & Alerts", "type": "dashboard"
      }]
  },
  "editable": true,
  "gnetId": null,
  "graphTooltip": 0,
  "id": 26,
  "links": [],
  "panels": [

{{ $alreadyGotError := false }}
{{ $foundAny := false }}
{{range $index, $metric := .Metrics }}
    {{if eq $metric.MetricType "counter" "gauge" }}
        {{ if $index }},{{end}}{{template "counter_gauge_cumulative" . }},
        {{template "counter_gauge_rate" . }}
    {{else if eq $metric.MetricType "errors"}}
        {{ if not $alreadyGotError }}
			{{ if $index }},{{end}}{{template "errors" . }}
			{{ $alreadyGotError = true }}
		{{ end }}
    {{else if eq $metric.MetricType "summary" "timer"}}
        {{ if $index }},{{end}}{{template "summary_timer" . }}
    {{end}}
    {{ $foundAny = true }}
{{end}}
{{range $metric := .ExternalTimers }}
	{{ if $foundAny }},{{end}}{{template "summary_timer" . }}
	{{ $foundAny = true }}
{{end}}

  ],
  "refresh": false,
  "schemaVersion": 16,
  "style": "dark",
  "tags": [{{ range $index, $element := .DashboardTags}}{{ if $index }}, {{end}}"{{ $element }}"{{end}}],
  "templating": {"list": []},
  "time": {"from": "now/d", "to": "now"},
  "timepicker": {
    "refresh_intervals": ["5s","10s","30s","1m","5m","15m","30m","1h","2h","1d"],
    "time_options": ["5m","15m","1h","6h","12h","24h","2d","7d","30d"]
  },
  "timezone": "",
  "title": "{{ .Title }}",
  "uid": "{{ .Id }}",
  "version": 1
}
`
