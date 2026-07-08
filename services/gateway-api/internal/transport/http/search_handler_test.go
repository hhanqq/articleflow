package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type fakeSearchProvider struct {
	candidates []parserv1.ArticleCandidate
}

func (provider fakeSearchProvider) Search(query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	return provider.candidates, nil
}

func TestSearchHandlerReturnsCandidates(t *testing.T) {
	handler := NewSearchHandler(fakeSearchProvider{
		candidates: []parserv1.ArticleCandidate{
			{
				SourceName:  "habr",
				ExternalID:  "habr-123",
				URL:         "https://habr.com/ru/articles/123/",
				Title:       "Go Kafka",
				PublishedAt: time.Date(2026, 7, 7, 10, 30, 0, 0, time.UTC),
			},
		},
	})
	body := bytes.NewBufferString(`{"query":"go kafka","sources":["habr"],"limit":10}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/search", body)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload SearchResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(payload.Candidates))
	}
	if payload.Limit != 10 || payload.Offset != 0 || payload.ReturnedCount != 1 {
		t.Fatalf("unexpected pagination metadata: %#v", payload)
	}
}
