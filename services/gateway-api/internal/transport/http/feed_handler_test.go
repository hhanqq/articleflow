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
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
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

type fakeUserReactionProvider struct {
	reactions []userv1.UserReaction
	userID    string
}

func (provider *fakeUserReactionProvider) ListUserReactions(_ context.Context, userID string) ([]userv1.UserReaction, error) {
	provider.userID = userID
	return provider.reactions, nil
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

func TestFeedHandlerPersonalizesQueryFeedByUserReactions(t *testing.T) {
	searcher := &fakeFeedSearchProvider{
		candidates: []parserv1.ArticleCandidate{
			{SourceName: "habr", ExternalID: "1", URL: "https://habr.com/1", Title: "Keep"},
			{SourceName: "vc", ExternalID: "2", URL: "https://vc.ru/2", Title: "Hide"},
			{SourceName: "dzen", ExternalID: "3", URL: "https://dzen.ru/3", Title: "Saved"},
		},
	}
	reactions := &fakeUserReactionProvider{
		reactions: []userv1.UserReaction{
			{UserID: "reader-1", ArticleID: "vc:2", Type: userv1.ReactionSkip, CreatedAt: time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)},
			{UserID: "reader-1", ArticleID: "dzen:3", Type: userv1.ReactionSave, CreatedAt: time.Date(2026, 7, 11, 10, 1, 0, 0, time.UTC)},
		},
	}
	handler := NewFeedHandlerWithRefill(FeedHandlerDependencies{
		FeedProvider:         fakeFeedProvider{},
		SearchProvider:       searcher,
		UserReactionProvider: reactions,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed?query=go&user_id=reader-1&limit=10", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if reactions.userID != "reader-1" {
		t.Fatalf("expected reader-1 reaction lookup, got %q", reactions.userID)
	}
	var payload FeedResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Items) != 2 {
		t.Fatalf("expected skipped item hidden, got %#v", payload.Items)
	}
	if payload.Items[0].ArticleID != "habr:1" || payload.Items[1].ArticleID != "dzen:3" {
		t.Fatalf("unexpected personalized items: %#v", payload.Items)
	}
	if !payload.Items[1].Saved || payload.Items[1].Reaction != string(userv1.ReactionSave) {
		t.Fatalf("expected saved marker on dzen item, got %#v", payload.Items[1])
	}
}

func TestFeedHandlerPassesDateFiltersToQueryFeed(t *testing.T) {
	searcher := &fakeFeedSearchProvider{}
	handler := NewFeedHandlerWithRefill(FeedHandlerDependencies{
		FeedProvider:   fakeFeedProvider{},
		SearchProvider: searcher,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed?query=новости&from_date=2026-07-01T00:00:00Z&to_date=2026-07-11T00:00:00Z", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if searcher.query.FromDate == nil || !searcher.query.FromDate.Equal(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected from date: %#v", searcher.query.FromDate)
	}
	if searcher.query.ToDate == nil || !searcher.query.ToDate.Equal(time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected to date: %#v", searcher.query.ToDate)
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
