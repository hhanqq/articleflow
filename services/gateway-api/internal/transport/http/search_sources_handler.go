package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type ParserSourceClient interface {
	ListSources(ctx context.Context) ([]parserv1.ParserSource, error)
	SetSourceEnabled(ctx context.Context, name string, enabled bool) (parserv1.ParserSource, bool, error)
}

type SearchSourcesResponse struct {
	Sources []parserv1.ParserSource `json:"sources"`
}

type SearchSourceResponse struct {
	Source parserv1.ParserSource `json:"source"`
}

type searchSourceUpdateRequest struct {
	Enabled bool `json:"enabled"`
}

func NewSearchSourcesHandler(client ParserSourceClient) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			sources, err := client.ListSources(request.Context())
			if err != nil {
				http.Error(response, "parser sources failed", http.StatusBadGateway)
				return
			}
			writeJSON(response, http.StatusOK, SearchSourcesResponse{Sources: sources})
		case http.MethodPatch:
			updateSearchSource(response, request, client)
		default:
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func updateSearchSource(response http.ResponseWriter, request *http.Request, client ParserSourceClient) {
	name := strings.TrimPrefix(request.URL.Path, "/api/v1/search/sources/")
	name = strings.TrimSpace(name)
	if name == "" || name == request.URL.Path {
		http.Error(response, "source name is required", http.StatusBadRequest)
		return
	}
	var payload searchSourceUpdateRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(response, "invalid json body", http.StatusBadRequest)
		return
	}
	source, ok, err := client.SetSourceEnabled(request.Context(), name, payload.Enabled)
	if err != nil {
		http.Error(response, "parser source update failed", http.StatusBadGateway)
		return
	}
	if !ok {
		http.Error(response, "source not found", http.StatusNotFound)
		return
	}
	writeJSON(response, http.StatusOK, SearchSourceResponse{Source: source})
}
