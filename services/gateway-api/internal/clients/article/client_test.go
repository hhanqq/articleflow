package article

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientGetsArticleByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/articles" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("id") != "habr:123" {
			t.Fatalf("unexpected id: %s", request.URL.Query().Get("id"))
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"article":{"ID":"habr:123","Title":"Go Kafka","Content":"Full text"}}`))
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	article, ok := client.GetByID("habr:123")

	if !ok {
		t.Fatal("expected article found")
	}
	if article.ID != "habr:123" {
		t.Fatalf("unexpected article id: %s", article.ID)
	}
}
