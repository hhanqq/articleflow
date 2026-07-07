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
