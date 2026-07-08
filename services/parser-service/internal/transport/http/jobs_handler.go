package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type JobManager interface {
	StartAsync(ctx context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error)
	Get(ctx context.Context, id string) (parserv1.ParserJob, bool, error)
	List(ctx context.Context, limit int) ([]parserv1.ParserJob, error)
}

type JobRequest struct {
	Query   string   `json:"query"`
	Sources []string `json:"sources"`
	Limit   int      `json:"limit"`
}

type JobResponse struct {
	Job parserv1.ParserJob `json:"job"`
}

type JobsResponse struct {
	Jobs []parserv1.ParserJob `json:"jobs"`
}

func NewJobsHandler(manager JobManager) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodPost:
			createJob(response, request, manager)
		case http.MethodGet:
			if request.URL.Path == "/api/v1/parser/jobs" {
				listJobs(response, request, manager)
				return
			}
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
	job, ok, err := manager.Get(request.Context(), id)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(response, "parser job not found", http.StatusNotFound)
		return
	}
	writeJSON(response, http.StatusOK, JobResponse{Job: job})
}

func listJobs(response http.ResponseWriter, request *http.Request, manager JobManager) {
	limit := 20
	if rawLimit := strings.TrimSpace(request.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(response, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}
	if limit > 100 {
		limit = 100
	}
	jobs, err := manager.List(request.Context(), limit)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(response, http.StatusOK, JobsResponse{Jobs: jobs})
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}
