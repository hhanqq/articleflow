package httptransport

import (
	"strconv"
	"strings"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	"github.com/hanq/articleflow/packages/observability"
)

type SourceMetrics interface {
	ObserveSourceEnabled(name string, enabled bool)
}

type SourceMetricsSink interface {
	AddLabels(name string, labels map[string]string, value float64)
	SetGaugeLabels(name string, labels map[string]string, value float64)
}

type sourceMetricsRecorder struct {
	metrics SourceMetricsSink
}

func NewObservabilitySourceMetrics(metrics *observability.MetricsRegistry) sourceMetricsRecorder {
	return sourceMetricsRecorder{metrics: metrics}
}

func (recorder sourceMetricsRecorder) ObserveSourceSnapshot(sources []parserv1.ParserSource) {
	for _, source := range sources {
		recorder.setSourceEnabledGauge(source.Name, source.Enabled)
	}
}

func (recorder sourceMetricsRecorder) ObserveSourceEnabled(name string, enabled bool) {
	if recorder.metrics == nil {
		return
	}
	name = cleanSourceMetricLabel(name)
	recorder.metrics.AddLabels("articleflow_parser_source_admin_updates_total", map[string]string{
		"source":  name,
		"enabled": strconv.FormatBool(enabled),
	}, 1)
	recorder.setSourceEnabledGauge(name, enabled)
}

func (recorder sourceMetricsRecorder) setSourceEnabledGauge(name string, enabled bool) {
	if recorder.metrics == nil {
		return
	}
	value := 0.0
	if enabled {
		value = 1
	}
	recorder.metrics.SetGaugeLabels("articleflow_parser_source_enabled", map[string]string{
		"source": cleanSourceMetricLabel(name),
	}, value)
}

func cleanSourceMetricLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
