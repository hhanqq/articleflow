package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hanq/articleflow/services/gateway-api/internal/config"
)

func TestHandlerExposesHealthRoute(t *testing.T) {
	handler := New(config.Config{ServiceName: "gateway-api", HTTPAddr: ":0"}).Handler()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
}
