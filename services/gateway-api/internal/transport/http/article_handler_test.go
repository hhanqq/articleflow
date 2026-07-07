package httptransport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
)

type fakeArticleProvider struct {
	article articlev1.Article
	ok      bool
}

func (provider fakeArticleProvider) GetByID(_ string) (articlev1.Article, bool) {
	return provider.article, provider.ok
}

func TestArticleHandlerReturnsArticle(t *testing.T) {
	handler := NewArticleHandler(fakeArticleProvider{
		article: articlev1.Article{ID: "habr:123", Title: "Go Kafka", Content: "Full text"},
		ok:      true,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/articles?id=habr%3A123", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload ArticleResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Article.ID != "habr:123" {
		t.Fatalf("unexpected article id: %s", payload.Article.ID)
	}
}

func TestArticleHandlerRequiresID(t *testing.T) {
	handler := NewArticleHandler(fakeArticleProvider{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/articles", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}
