package httptransport

import (
	"encoding/json"
	"net/http"
	"strings"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
)

type ArticleProvider interface {
	GetByID(id string) (articlev1.Article, bool)
}

type ArticleResponse struct {
	Article articlev1.Article `json:"article"`
}

func NewArticleHandler(provider ArticleProvider) http.Handler {
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
		article, ok := provider.GetByID(id)
		if !ok {
			http.Error(response, "article not found", http.StatusNotFound)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(ArticleResponse{Article: article})
	})
}
