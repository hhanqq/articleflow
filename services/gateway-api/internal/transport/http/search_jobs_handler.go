package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type ParserJobClient interface {
	StartAsync(ctx context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error)
	Get(ctx context.Context, id string) (parserv1.ParserJob, bool, error)
}

type SearchJobRequest struct {
	Query   string   `json:"query"`
	Sources []string `json:"sources"`
	Limit   int      `json:"limit"`
}

type SearchJobResponse struct {
	Job parserv1.ParserJob `json:"job"`
}

func NewSearchJobsHandler(client ParserJobClient) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodPost:
			startSearchJob(response, request, client)
		case http.MethodGet:
			getSearchJob(response, request, client)
		default:
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func startSearchJob(response http.ResponseWriter, request *http.Request, client ParserJobClient) {
	var payload SearchJobRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(response, "invalid json body", http.StatusBadRequest)
		return
	}
	query := parserv1.SearchQuery{
		Text:    strings.TrimSpace(payload.Query),
		Sources: payload.Sources,
		Limit:   payload.Limit,
	}
	job, err := client.StartAsync(request.Context(), query)
	if err != nil {
		http.Error(response, "parser job failed", http.StatusBadGateway)
		return
	}
	writeJSON(response, http.StatusAccepted, SearchJobResponse{Job: job})
}

func getSearchJob(response http.ResponseWriter, request *http.Request, client ParserJobClient) {
	id := strings.TrimPrefix(request.URL.Path, "/api/v1/search/jobs/")
	id = strings.TrimSpace(id)
	if id == "" || id == request.URL.Path {
		http.Error(response, "job id is required", http.StatusBadRequest)
		return
	}
	job, ok, err := client.Get(request.Context(), id)
	if err != nil {
		http.Error(response, "parser job failed", http.StatusBadGateway)
		return
	}
	if !ok {
		http.Error(response, "parser job not found", http.StatusNotFound)
		return
	}
	writeJSON(response, http.StatusOK, SearchJobResponse{Job: job})
}
