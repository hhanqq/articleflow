package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type MetricsRegistry struct {
	mu       sync.RWMutex
	counters map[string]float64
}

func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{counters: make(map[string]float64)}
}

func (registry *MetricsRegistry) Inc(name string) {
	registry.Add(name, 1)
}

func (registry *MetricsRegistry) Add(name string, value float64) {
	registry.AddLabels(name, nil, value)
}

func (registry *MetricsRegistry) AddLabels(name string, labels map[string]string, value float64) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.counters[metricKey(name, labels)] += value
}

func (registry *MetricsRegistry) Snapshot() map[string]float64 {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	snapshot := make(map[string]float64, len(registry.counters))
	for name, value := range registry.counters {
		metricName, labels := parseMetricKey(name)
		snapshot[renderMetricName(metricName, labels)] = value
	}
	return snapshot
}

func NewPrometheusHandler(serviceName string, registry *MetricsRegistry) http.Handler {
	if registry == nil {
		registry = NewMetricsRegistry()
	}
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		snapshot := registry.Snapshot()
		names := make([]string, 0, len(snapshot))
		for name := range snapshot {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			metricName, labels := parseRenderedMetricName(name)
			labels = append([]label{{name: "service", value: serviceName}}, labels...)
			_, _ = fmt.Fprintf(response, `%s%s %g`+"\n", metricName, renderLabels(labels), snapshot[name])
		}
	})
}

func InstrumentHTTPRequests(registry *MetricsRegistry, next http.Handler) http.Handler {
	if registry == nil {
		registry = NewMetricsRegistry()
	}
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		registry.Inc("articleflow_http_requests_total")
		next.ServeHTTP(response, request)
	})
}

type label struct {
	name  string
	value string
}

func metricKey(name string, labels map[string]string) string {
	cleanLabels := cleanLabels(labels)
	if len(cleanLabels) == 0 {
		return name
	}
	parts := make([]string, 0, len(cleanLabels))
	for _, item := range cleanLabels {
		parts = append(parts, item.name+"="+item.value)
	}
	return name + "\x00" + strings.Join(parts, "\x00")
}

func parseMetricKey(key string) (string, []label) {
	parts := strings.Split(key, "\x00")
	if len(parts) == 1 {
		return key, nil
	}
	labels := make([]label, 0, len(parts)-1)
	for _, part := range parts[1:] {
		name, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		labels = append(labels, label{name: name, value: value})
	}
	return parts[0], labels
}

func renderMetricName(name string, labels []label) string {
	return name + renderLabels(labels)
}

func parseRenderedMetricName(value string) (string, []label) {
	name, rawLabels, ok := strings.Cut(value, "{")
	if !ok {
		return value, nil
	}
	rawLabels = strings.TrimSuffix(rawLabels, "}")
	parts := strings.Split(rawLabels, ",")
	labels := make([]label, 0, len(parts))
	for _, part := range parts {
		labelName, labelValue, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		labels = append(labels, label{name: labelName, value: strings.Trim(labelValue, `"`)})
	}
	return name, labels
}

func renderLabels(labels []label) string {
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labels))
	for _, item := range labels {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, item.name, escapeLabelValue(item.value)))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func cleanLabels(labels map[string]string) []label {
	if len(labels) == 0 {
		return nil
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	result := make([]label, 0, len(keys))
	for _, key := range keys {
		result = append(result, label{name: key, value: strings.TrimSpace(labels[key])})
	}
	return result
}

func escapeLabelValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}
