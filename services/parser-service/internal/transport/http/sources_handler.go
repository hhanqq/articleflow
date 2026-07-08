package httptransport

import (
	"net/http"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type SourcesResponse struct {
	Sources []parserv1.ParserSource `json:"sources"`
}

func NewSourcesHandler(sources []parserv1.ParserSource) http.Handler {
	copied := append([]parserv1.ParserSource(nil), sources...)
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(response, http.StatusOK, SourcesResponse{Sources: copied})
	})
}
