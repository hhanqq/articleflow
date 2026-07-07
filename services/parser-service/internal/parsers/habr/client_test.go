package habr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type recordingWaiter struct {
	calls int
}

func (waiter *recordingWaiter) Wait(context.Context) error {
	waiter.calls++
	return nil
}

func TestClientSearchFetchesSearchAndFullArticle(t *testing.T) {
	searchRSS, err := os.ReadFile("testdata/search_rss.xml")
	if err != nil {
		t.Fatalf("read search fixture: %v", err)
	}
	articleHTML, err := os.ReadFile("testdata/article.html")
	if err != nil {
		t.Fatalf("read article fixture: %v", err)
	}
	var searchQuery string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/ru/rss/search/":
			searchQuery = request.URL.Query().Get("q")
			_, _ = response.Write(searchRSS)
		case "/ru/articles/900001/":
			_, _ = response.Write(articleHTML)
		case "/ru/articles/900002/":
			_, _ = response.Write(articleHTML)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()
	waiter := &recordingWaiter{}
	client := NewClient(ClientOptions{
		BaseURL:     server.URL,
		HTTPClient:  server.Client(),
		MaxAttempts: 1,
		Waiter:      waiter,
	})

	candidates, err := client.Search(context.Background(), parserv1.SearchQuery{Text: "go kafka", Limit: 1})

	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if searchQuery != "go kafka" {
		t.Fatalf("unexpected search query: %s", searchQuery)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Content == "" {
		t.Fatal("expected full article content")
	}
	if waiter.calls != 2 {
		t.Fatalf("expected waiter before search and article requests, got %d", waiter.calls)
	}
}

func TestClientSearchRetriesTemporaryHTTPFailure(t *testing.T) {
	searchRSS, err := os.ReadFile("testdata/search_rss.xml")
	if err != nil {
		t.Fatalf("read search fixture: %v", err)
	}
	articleHTML, err := os.ReadFile("testdata/article.html")
	if err != nil {
		t.Fatalf("read article fixture: %v", err)
	}
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/ru/rss/search/" {
			_, _ = response.Write(articleHTML)
			return
		}
		attempts++
		if attempts == 1 {
			http.Error(response, "temporary", http.StatusBadGateway)
			return
		}
		_, _ = response.Write(searchRSS)
	}))
	defer server.Close()
	client := NewClient(ClientOptions{
		BaseURL:     server.URL,
		HTTPClient:  server.Client(),
		MaxAttempts: 2,
		Waiter:      NoopWaiter{},
	})

	_, err = client.Search(context.Background(), parserv1.SearchQuery{Text: "go kafka", Limit: 0})

	if err != nil {
		t.Fatalf("expected retry to recover, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}
