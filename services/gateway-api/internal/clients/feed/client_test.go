package feed

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientListsFeedItemsFromFeedService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/feed" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("limit") != "7" {
			t.Fatalf("unexpected limit: %s", request.URL.Query().Get("limit"))
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"items":[{"ArticleID":"article-1","Title":"Go Kafka","Score":44}]}`))
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	items := client.List(7)

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ArticleID != "article-1" {
		t.Fatalf("unexpected article id: %s", items[0].ArticleID)
	}
}
