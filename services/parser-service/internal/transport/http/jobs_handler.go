package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type JobManager interface {
	StartAsync(ctx context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error)
	Get(id string) (parserv1.ParserJob, bool)
}

type JobRequest struct {
	Query   string   `json:"query"`
	Sources []string `json:"sources"`
	Limit   int      `json:"limit"`
}

type JobResponse struct {
	Job parserv1.ParserJob `json:"job"`
}

func NewJobsHandler(manager JobManager) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodPost:
			createJob(response, request, manager)
		case http.MethodGet:
			getJob(response, request, manager)
		default:
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func createJob(response http.ResponseWriter, request *http.Request, manager JobManager) {
	var payload JobRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(response, "invalid json body", http.StatusBadRequest)
		return
	}
	query := parserv1.SearchQuery{
		Text:    strings.TrimSpace(payload.Query),
		Sources: payload.Sources,
		Limit:   payload.Limit,
	}
	job, err := manager.StartAsync(request.Context(), query)
	if err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(response, http.StatusAccepted, JobResponse{Job: job})
}

func getJob(response http.ResponseWriter, request *http.Request, manager JobManager) {
	id := strings.TrimPrefix(request.URL.Path, "/api/v1/parser/jobs/")
	id = strings.TrimSpace(id)
	if id == "" || id == request.URL.Path {
		http.Error(response, "job id is required", http.StatusBadRequest)
		return
	}
	job, ok := manager.Get(id)
	if !ok {
		http.Error(response, "parser job not found", http.StatusNotFound)
		return
	}
	writeJSON(response, http.StatusOK, JobResponse{Job: job})
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}
