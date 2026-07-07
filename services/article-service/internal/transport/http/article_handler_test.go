package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
	"github.com/hanq/articleflow/services/article-service/internal/usecase"
)

func TestArticleHandlerReturnsArticleByID(t *testing.T) {
	store := usecase.NewMemoryArticleStore()
	article := articlev1.Article{
		ID:          "habr:123",
		SourceName:  "habr",
		ExternalID:  "123",
		URL:         "https://habr.com/ru/articles/123/",
		Title:       "Go Kafka",
		Content:     "Full article text",
		PublishedAt: time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
		ParsedAt:    time.Date(2026, 7, 8, 10, 5, 0, 0, time.UTC),
	}
	if _, err := store.Save(context.Background(), article); err != nil {
		t.Fatalf("save article: %v", err)
	}
	handler := NewArticleHandler(store)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/articles?id=habr%3A123", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload ArticleResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Article.ID != "habr:123" {
		t.Fatalf("unexpected article id: %s", payload.Article.ID)
	}
	if payload.Article.Content != "Full article text" {
		t.Fatalf("unexpected article content: %s", payload.Article.Content)
	}
}

func TestArticleHandlerReturnsNotFound(t *testing.T) {
	handler := NewArticleHandler(usecase.NewMemoryArticleStore())
	request := httptest.NewRequest(http.MethodGet, "/api/v1/articles?id=missing", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}
