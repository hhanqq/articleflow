package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitMiddlewareLimitsRequestsByClientIP(t *testing.T) {
	handler := NewRateLimitMiddleware(2, time.Minute, http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))
	request := func() *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/feed", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		handler.ServeHTTP(response, req)
		return response
	}

	if request().Code != http.StatusOK {
		t.Fatal("expected first request to pass")
	}
	if request().Code != http.StatusOK {
		t.Fatal("expected second request to pass")
	}
	if response := request(); response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected third request to be rate limited, got %d", response.Code)
	}
}
