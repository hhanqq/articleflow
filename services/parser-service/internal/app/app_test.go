package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hanq/articleflow/services/parser-service/internal/config"
)

func TestHandlerExposesParserJobsRoute(t *testing.T) {
	handler := New(config.Config{
		ServiceName:        "parser-service",
		HTTPAddr:           ":0",
		KafkaBrokers:       "localhost:9092",
		HabrBaseURL:        "https://habr.com",
		HabrMaxAttempts:    1,
		HabrRequestDelayMS: 0,
	}).Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/parser/jobs/missing", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}

func TestHandlerExposesHealthRoute(t *testing.T) {
	handler := New(config.Config{
		ServiceName:        "parser-service",
		HTTPAddr:           ":0",
		KafkaBrokers:       "localhost:9092",
		HabrBaseURL:        "https://habr.com",
		HabrMaxAttempts:    1,
		HabrRequestDelayMS: 0,
	}).Handler()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
}
