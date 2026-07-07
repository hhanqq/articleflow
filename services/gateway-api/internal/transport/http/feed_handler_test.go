package httptransport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
)

type fakeFeedProvider struct {
	items []feedv1.FeedItem
}

func (provider fakeFeedProvider) List(limit int) []feedv1.FeedItem {
	return provider.items
}

func TestFeedHandlerReturnsJSONFeed(t *testing.T) {
	handler := NewFeedHandler(fakeFeedProvider{
		items: []feedv1.FeedItem{
			{
				ArticleID:   "article-1",
				Title:       "Go microservices",
				SourceName:  "habr",
				URL:         "https://habr.com/ru/articles/123/",
				PublishedAt: time.Date(2026, 7, 7, 10, 30, 0, 0, time.UTC),
			},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed?limit=10", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload FeedResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON response, got error: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	if payload.Items[0].ArticleID != "article-1" {
		t.Fatalf("unexpected article id: %s", payload.Items[0].ArticleID)
	}
}
