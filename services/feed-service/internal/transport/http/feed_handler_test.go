package httptransport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
)

type fakeFeedProvider struct {
	items []feedv1.FeedItem
}

func (provider fakeFeedProvider) List(_ int) ([]feedv1.FeedItem, error) {
	return provider.items, nil
}

func TestFeedHandlerReturnsItems(t *testing.T) {
	handler := NewFeedHandler(fakeFeedProvider{items: []feedv1.FeedItem{{ArticleID: "article-1", Title: "Go"}}})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed?limit=10", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload FeedResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
}
