package jobs

import (
	"strings"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	"github.com/hanq/articleflow/packages/observability"
)

type MetricsRecorder interface {
	ObserveParserJob(job parserv1.ParserJob)
}

type MetricsSink interface {
	AddLabels(name string, labels map[string]string, value float64)
}

type prometheusMetricsRecorder struct {
	metrics MetricsSink
}

func NewMetricsRecorder(metrics MetricsSink) MetricsRecorder {
	if metrics == nil {
		return nil
	}
	return prometheusMetricsRecorder{metrics: metrics}
}

func NewObservabilityMetricsRecorder(metrics *observability.MetricsRegistry) MetricsRecorder {
	return NewMetricsRecorder(metrics)
}

func (recorder prometheusMetricsRecorder) ObserveParserJob(job parserv1.ParserJob) {
	status := cleanMetricLabel(job.Status, "unknown")
	recorder.metrics.AddLabels("articleflow_parser_jobs_total", map[string]string{
		"status": status,
	}, 1)
	for _, stat := range job.SourceStats {
		labels := map[string]string{
			"source":   cleanMetricLabel(stat.SourceName, "unknown"),
			"strategy": cleanMetricLabel(stat.Strategy, "unknown"),
			"status":   cleanMetricLabel(stat.Status, "unknown"),
		}
		recorder.metrics.AddLabels("articleflow_parser_source_found_total", labels, float64(stat.FoundCount))
		recorder.metrics.AddLabels("articleflow_parser_source_accepted_total", labels, float64(stat.AcceptedCount))
		recorder.metrics.AddLabels("articleflow_parser_source_returned_total", labels, float64(stat.ReturnedCount))
		recorder.metrics.AddLabels("articleflow_parser_source_published_total", labels, float64(stat.PublishedCount))
		recorder.metrics.AddLabels("articleflow_parser_source_filtered_total", labels, float64(stat.FilteredCount))
		recorder.metrics.AddLabels("articleflow_parser_source_duration_ms_total", labels, float64(stat.DurationMS))
	}
}

func cleanMetricLabel(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
