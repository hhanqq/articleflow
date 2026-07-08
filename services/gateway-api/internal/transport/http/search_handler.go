package httptransport

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type SearchProvider interface {
	Search(query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error)
}

type SearchRequest struct {
	Query    string     `json:"query"`
	Sources  []string   `json:"sources"`
	Tags     []string   `json:"tags"`
	FromDate *time.Time `json:"from_date"`
	ToDate   *time.Time `json:"to_date"`
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
}

type SearchResponse struct {
	Candidates    []parserv1.ArticleCandidate `json:"candidates"`
	Limit         int                         `json:"limit"`
	Offset        int                         `json:"offset"`
	ReturnedCount int                         `json:"returned_count"`
}

func NewSearchHandler(provider SearchProvider) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload SearchRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(response, "invalid json body", http.StatusBadRequest)
			return
		}
		query := parserv1.SearchQuery{
			Text:     strings.TrimSpace(payload.Query),
			Sources:  payload.Sources,
			Tags:     payload.Tags,
			FromDate: payload.FromDate,
			ToDate:   payload.ToDate,
			Limit:    normalizeSearchLimit(payload.Limit),
			Offset:   payload.Offset,
		}
		if err := query.Validate(); err != nil {
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}

		candidates, err := provider.Search(query)
		if err != nil {
			http.Error(response, "search failed", http.StatusBadGateway)
			return
		}

		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(SearchResponse{
			Candidates:    candidates,
			Limit:         query.Normalize().Limit,
			Offset:        query.Normalize().Offset,
			ReturnedCount: len(candidates),
		})
	})
}

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}
