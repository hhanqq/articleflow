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

func TestPrometheusHandlerRendersLabelledCounters(t *testing.T) {
	metrics := NewMetricsRegistry()
	metrics.AddLabels("articleflow_parser_source_found_total", map[string]string{
		"source": "habr",
		"status": "ok",
	}, 3)
	handler := NewPrometheusHandler("parser-service", metrics)
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	body := response.Body.String()
	if !strings.Contains(body, `articleflow_parser_source_found_total{service="parser-service",source="habr",status="ok"} 3`) {
		t.Fatalf("expected labelled source metric, got %q", body)
	}
}

func TestInstrumentHTTPRequestsCountsRequests(t *testing.T) {
	metrics := NewMetricsRegistry()
	handler := InstrumentHTTPRequests(metrics, http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusAccepted)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", response.Code)
	}
	snapshot := metrics.Snapshot()
	if snapshot["articleflow_http_requests_total"] != 1 {
		t.Fatalf("expected 1 counted request, got %g", snapshot["articleflow_http_requests_total"])
	}
}
