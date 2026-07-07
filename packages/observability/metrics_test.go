package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsRegistryCountsValues(t *testing.T) {
	metrics := NewMetricsRegistry()
	metrics.Inc("http_requests_total")
	metrics.Add("http_requests_total", 2)

	snapshot := metrics.Snapshot()

	if snapshot["http_requests_total"] != 3 {
		t.Fatalf("expected counter 3, got %f", snapshot["http_requests_total"])
	}
}

func TestPrometheusHandlerRendersCounters(t *testing.T) {
	metrics := NewMetricsRegistry()
	metrics.Inc("articleflow_requests_total")
	handler := NewPrometheusHandler("gateway-api", metrics)
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, `service="gateway-api"`) {
		t.Fatalf("expected service label in metrics, got %q", body)
	}
	if !strings.Contains(body, "articleflow_requests_total") {
		t.Fatalf("expected counter in metrics, got %q", body)
	}
}
