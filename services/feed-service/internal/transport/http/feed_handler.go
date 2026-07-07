package httptransport

import (
	"encoding/json"
	"net/http"
	"strconv"

	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
)

type FeedProvider interface {
	List(limit int) ([]feedv1.FeedItem, error)
}

type FeedResponse struct {
	Items []feedv1.FeedItem `json:"items"`
}

func NewFeedHandler(provider FeedProvider) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseLimit(request)
		items, err := provider.List(limit)
		if err != nil {
			http.Error(response, "feed failed", http.StatusBadGateway)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(FeedResponse{Items: items})
	})
}

func parseLimit(request *http.Request) int {
	limit, err := strconv.Atoi(request.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}
