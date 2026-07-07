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
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.counters[name] += value
}

func (registry *MetricsRegistry) Snapshot() map[string]float64 {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	snapshot := make(map[string]float64, len(registry.counters))
	for name, value := range registry.counters {
		snapshot[name] = value
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
			_, _ = fmt.Fprintf(response, `%s{service="%s"} %g`+"\n", name, serviceName, snapshot[name])
		}
	})
}
