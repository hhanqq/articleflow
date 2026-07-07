package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hanq/articleflow/services/feed-service/internal/config"
)

func TestHandlerExposesFeedRoute(t *testing.T) {
	handler := New(config.Config{ServiceName: "feed-service", HTTPAddr: ":0"}).Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
}
