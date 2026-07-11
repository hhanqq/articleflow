package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
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

type fakeFeedSearchProvider struct {
	candidates []parserv1.ArticleCandidate
	query      parserv1.SearchQuery
	calls      int
}

func (provider *fakeFeedSearchProvider) Search(query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	provider.query = query
	provider.calls++
	return provider.candidates, nil
}

type fakeFeedRefillStarter struct {
	job     parserv1.ParserJob
	started []parserv1.SearchQuery
}

func (starter *fakeFeedRefillStarter) StartAsync(_ context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error) {
	starter.started = append(starter.started, query)
	return starter.job, nil
}

func TestFeedHandlerSearchesStoredArticlesForQueryFeed(t *testing.T) {
	searcher := &fakeFeedSearchProvider{
		candidates: []parserv1.ArticleCandidate{
			{
				SourceName:  "dzen",
				ExternalID:  "dzen-1",
				URL:         "https://dzen.ru/a/1",
				Title:       "Поездка в Турцию",
				Summary:     "Маршрут и расходы",
				Tags:        []string{"travel"},
				PublishedAt: time.Date(2026, 7, 7, 10, 30, 0, 0, time.UTC),
			},
		},
	}
	handler := NewFeedHandlerWithRefill(FeedHandlerDependencies{
		FeedProvider:   fakeFeedProvider{},
		SearchProvider: searcher,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed?query=турция&sources=dzen,habr&limit=10", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload FeedResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Mode != "query" {
		t.Fatalf("expected query mode, got %q", payload.Mode)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	if payload.Items[0].Title != "Поездка в Турцию" {
		t.Fatalf("unexpected title: %s", payload.Items[0].Title)
	}
	if searcher.query.Text != "турция" || searcher.query.Limit != 10 {
		t.Fatalf("unexpected search query: %#v", searcher.query)
	}
	if len(searcher.query.Sources) != 2 || searcher.query.Sources[0] != "dzen" || searcher.query.Sources[1] != "habr" {
		t.Fatalf("unexpected sources: %#v", searcher.query.Sources)
	}
}

func TestFeedHandlerStartsBackgroundRefillWhenQueryFeedIsThin(t *testing.T) {
	searcher := &fakeFeedSearchProvider{}
	refiller := &fakeFeedRefillStarter{job: parserv1.ParserJob{
		ID:     "parser-job-refill",
		Status: parserv1.ParserJobStatusQueued,
	}}
	handler := NewFeedHandlerWithRefill(FeedHandlerDependencies{
		FeedProvider:    fakeFeedProvider{},
		SearchProvider:  searcher,
		ParserJobClient: refiller,
		RefillMinItems:  5,
		RefillLimit:     80,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed?query=go+kafka&sources=habr,dzen&limit=20&refill=true", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload FeedResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !payload.RefillStarted {
		t.Fatal("expected background refill to start")
	}
	if payload.RefillJob == nil || payload.RefillJob.ID != "parser-job-refill" {
		t.Fatalf("unexpected refill job: %#v", payload.RefillJob)
	}
	if len(refiller.started) != 1 {
		t.Fatalf("expected one refill job, got %d", len(refiller.started))
	}
	started := refiller.started[0]
	if started.Text != "go kafka" || started.Limit != 80 {
		t.Fatalf("unexpected refill query: %#v", started)
	}
	if len(started.Sources) != 2 || started.Sources[0] != "habr" || started.Sources[1] != "dzen" {
		t.Fatalf("unexpected refill sources: %#v", started.Sources)
	}
}

func TestFeedHandlerUsesCursorForQueryFeedPagination(t *testing.T) {
	searcher := &fakeFeedSearchProvider{
		candidates: []parserv1.ArticleCandidate{
			{
				SourceName: "habr",
				ExternalID: "habr-2",
				URL:        "https://habr.com/ru/articles/2/",
				Title:      "Go Kafka page two",
			},
		},
	}
	handler := NewFeedHandlerWithRefill(FeedHandlerDependencies{
		FeedProvider:   fakeFeedProvider{},
		SearchProvider: searcher,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed?query=go+kafka&limit=1&cursor=offset:1", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload FeedResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if searcher.query.Offset != 1 {
		t.Fatalf("expected search offset from cursor, got %#v", searcher.query)
	}
	if payload.NextCursor != "offset:2" {
		t.Fatalf("expected next cursor offset:2, got %q", payload.NextCursor)
	}
}

func TestFeedHandlerCachesQueryFeedResponses(t *testing.T) {
	searcher := &fakeFeedSearchProvider{
		candidates: []parserv1.ArticleCandidate{
			{
				SourceName: "habr",
				ExternalID: "habr-1",
				URL:        "https://habr.com/ru/articles/1/",
				Title:      "Go Kafka cache",
			},
		},
	}
	handler := NewFeedHandlerWithRefill(FeedHandlerDependencies{
		FeedProvider:   fakeFeedProvider{},
		SearchProvider: searcher,
		CacheTTL:       time.Minute,
	})

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v1/feed?query=go+kafka&limit=1", nil))
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/v1/feed?query=go+kafka&limit=1", nil))

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("expected 200 responses, got %d and %d", first.Code, second.Code)
	}
	if searcher.calls != 1 {
		t.Fatalf("expected cached second response, search calls=%d", searcher.calls)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("expected identical cached response")
	}
}
