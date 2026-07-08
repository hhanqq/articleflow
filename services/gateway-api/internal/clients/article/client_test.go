package article

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
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

func TestClientSearchesArticles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/articles/search" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", request.Method)
		}
		var payload searchRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Query != "путешествие в китай" {
			t.Fatalf("unexpected query: %s", payload.Query)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"articles":[{"ID":"vc:1","SourceName":"vc","ExternalID":"1","URL":"https://vc.ru/1","Title":"Путешествие в Китай","Summary":"Маршрут"}]}`))
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	candidates, err := client.Search(parserv1.SearchQuery{Text: "путешествие в китай", Sources: []string{"vc"}, Limit: 5})

	if err != nil {
		t.Fatalf("search articles: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].SourceName != "vc" || candidates[0].Title != "Путешествие в Китай" {
		t.Fatalf("unexpected candidate: %#v", candidates[0])
	}
}
