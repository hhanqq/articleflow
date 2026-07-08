package httptransport

import (
	"context"
	"net/http"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type ParserSourceClient interface {
	ListSources(ctx context.Context) ([]parserv1.ParserSource, error)
}

type SearchSourcesResponse struct {
	Sources []parserv1.ParserSource `json:"sources"`
}

func NewSearchSourcesHandler(client ParserSourceClient) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		sources, err := client.ListSources(request.Context())
		if err != nil {
			http.Error(response, "parser sources failed", http.StatusBadGateway)
			return
		}
		writeJSON(response, http.StatusOK, SearchSourcesResponse{Sources: sources})
	})
}
