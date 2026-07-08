package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
)

type ArticleReader interface {
	GetByID(ctx context.Context, id string) (articlev1.Article, bool, error)
}

type ArticleSearcher interface {
	Search(ctx context.Context, query articlev1.SearchQuery) ([]articlev1.Article, error)
}

type ArticleResponse struct {
	Article articlev1.Article `json:"article"`
}

type ArticleSearchRequest struct {
	Query    string     `json:"query"`
	Sources  []string   `json:"sources"`
	Tags     []string   `json:"tags"`
	FromDate *time.Time `json:"from_date"`
	ToDate   *time.Time `json:"to_date"`
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
}

type ArticleSearchResponse struct {
	Articles      []articlev1.Article `json:"articles"`
	Limit         int                 `json:"limit"`
	Offset        int                 `json:"offset"`
	ReturnedCount int                 `json:"returned_count"`
}

func NewArticleHandler(reader ArticleReader) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimSpace(request.URL.Query().Get("id"))
		if id == "" {
			http.Error(response, "id is required", http.StatusBadRequest)
			return
		}
		article, ok, err := reader.GetByID(request.Context(), id)
		if err != nil {
			http.Error(response, err.Error(), http.StatusBadGateway)
			return
		}
		if !ok {
			http.Error(response, "article not found", http.StatusNotFound)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(ArticleResponse{Article: article})
	})
}

func NewArticleSearchHandler(searcher ArticleSearcher) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload ArticleSearchRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(response, "invalid json body", http.StatusBadRequest)
			return
		}
		query := articlev1.SearchQuery{
			Text:     strings.TrimSpace(payload.Query),
			Sources:  payload.Sources,
			Tags:     payload.Tags,
			FromDate: payload.FromDate,
			ToDate:   payload.ToDate,
			Limit:    payload.Limit,
			Offset:   payload.Offset,
		}.Normalize()
		if err := query.Validate(); err != nil {
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}
		articles, err := searcher.Search(request.Context(), query)
		if err != nil {
			http.Error(response, err.Error(), http.StatusBadGateway)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(ArticleSearchResponse{
			Articles:      articles,
			Limit:         query.Limit,
			Offset:        query.Offset,
			ReturnedCount: len(articles),
		})
	})
}
