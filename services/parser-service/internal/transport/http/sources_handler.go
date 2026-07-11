package httptransport

import (
	"encoding/json"
	"net/http"
	"strings"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type SourceRegistry interface {
	List() []parserv1.ParserSource
	SetEnabled(name string, enabled bool) (parserv1.ParserSource, bool)
}

type SourcesResponse struct {
	Sources []parserv1.ParserSource `json:"sources"`
}

type SourceResponse struct {
	Source parserv1.ParserSource `json:"source"`
}

type sourceUpdateRequest struct {
	Enabled bool `json:"enabled"`
}

func NewSourcesHandler(registry SourceRegistry) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			writeJSON(response, http.StatusOK, SourcesResponse{Sources: registry.List()})
		case http.MethodPatch:
			updateSource(response, request, registry)
		default:
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func updateSource(response http.ResponseWriter, request *http.Request, registry SourceRegistry) {
	name := strings.TrimPrefix(request.URL.Path, "/api/v1/parser/sources/")
	name = strings.TrimSpace(name)
	if name == "" || name == request.URL.Path {
		http.Error(response, "source name is required", http.StatusBadRequest)
		return
	}
	var payload sourceUpdateRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(response, "invalid json body", http.StatusBadRequest)
		return
	}
	source, ok := registry.SetEnabled(name, payload.Enabled)
	if !ok {
		http.Error(response, "source not found", http.StatusNotFound)
		return
	}
	writeJSON(response, http.StatusOK, SourceResponse{Source: source})
}
