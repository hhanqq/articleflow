package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouterExposesHealthRoute(t *testing.T) {
	router := NewRouter(RouterDependencies{
		ServiceName:      "gateway-api",
		FeedProvider:     fakeFeedProvider{},
		SearchProvider:   fakeSearchProvider{},
		ReactionRecorder: &fakeReactionRecorder{},
	})
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
}

func TestRouterAddsCORSHeadersForSPA(t *testing.T) {
	router := NewRouter(RouterDependencies{
		ServiceName:      "gateway-api",
		FeedProvider:     fakeFeedProvider{},
		SearchProvider:   fakeSearchProvider{},
		ReactionRecorder: &fakeReactionRecorder{},
	})
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/feed", nil)
	request.Header.Set("Origin", "http://127.0.0.1:5173")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:5173" {
		t.Fatalf("missing CORS origin header")
	}
}

func TestNewRouterExposesMetricsRoute(t *testing.T) {
	router := NewRouter(RouterDependencies{
		ServiceName:      "gateway-api",
		FeedProvider:     fakeFeedProvider{},
		SearchProvider:   fakeSearchProvider{},
		ReactionRecorder: &fakeReactionRecorder{},
	})
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if response.Body.String() == "" {
		t.Fatal("expected metrics body")
	}
}
