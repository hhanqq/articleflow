package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type fakeUserProfileProvider struct {
	profile userv1.UserProfile
	userID  string
}

func (provider *fakeUserProfileProvider) EnsureUserProfile(_ context.Context, userID string) (userv1.UserProfile, error) {
	provider.userID = userID
	if provider.profile.ID == "" {
		provider.profile.ID = userID
	}
	return provider.profile, nil
}

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

func TestRouterCreatesAnonymousSessionAndExposesMe(t *testing.T) {
	profiles := &fakeUserProfileProvider{}
	router := NewRouter(RouterDependencies{
		ServiceName:         "gateway-api",
		FeedProvider:        fakeFeedProvider{},
		SearchProvider:      fakeSearchProvider{},
		UserProfileProvider: profiles,
		ReactionRecorder:    &fakeReactionRecorder{},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	cookie := response.Result().Cookies()[0]
	if cookie.Name != UserIDCookieName || cookie.Value == "" {
		t.Fatalf("expected anonymous user cookie, got %#v", cookie)
	}
	if profiles.userID != cookie.Value {
		t.Fatalf("expected profile ensure for cookie user id, got %q and cookie %q", profiles.userID, cookie.Value)
	}
	var payload MeResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.User.ID != cookie.Value {
		t.Fatalf("expected me response to use session id, got %#v", payload.User)
	}
}
