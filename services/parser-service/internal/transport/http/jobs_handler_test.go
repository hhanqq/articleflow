package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type fakeJobManager struct {
	job parserv1.ParserJob
}

func (manager fakeJobManager) StartAsync(_ context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error) {
	manager.job.Query = query
	return manager.job, nil
}

func (manager fakeJobManager) Get(id string) (parserv1.ParserJob, bool) {
	if manager.job.ID == id {
		return manager.job, true
	}
	return parserv1.ParserJob{}, false
}

func TestJobsHandlerCreatesParserJob(t *testing.T) {
	handler := NewJobsHandler(fakeJobManager{
		job: parserv1.ParserJob{
			ID:        "parser-job-1",
			Status:    parserv1.ParserJobStatusQueued,
			CreatedAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/parser/jobs", bytes.NewBufferString(`{"query":"go kafka","sources":["habr"],"limit":5}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", response.Code)
	}
	var payload JobResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Job.ID != "parser-job-1" {
		t.Fatalf("unexpected job id: %s", payload.Job.ID)
	}
}

func TestJobsHandlerReturnsParserJobStatus(t *testing.T) {
	handler := NewJobsHandler(fakeJobManager{
		job: parserv1.ParserJob{
			ID:              "parser-job-1",
			Status:          parserv1.ParserJobStatusCompleted,
			CandidatesCount: 1,
			Candidates: []parserv1.ArticleCandidate{
				{SourceName: "habr", ExternalID: "1", Title: "Go Kafka"},
			},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/parser/jobs/parser-job-1", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload JobResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Job.Status != parserv1.ParserJobStatusCompleted {
		t.Fatalf("unexpected status: %s", payload.Job.Status)
	}
	if len(payload.Job.Candidates) != 1 {
		t.Fatalf("expected candidates in response, got %d", len(payload.Job.Candidates))
	}
}
