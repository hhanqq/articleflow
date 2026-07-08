package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type fakeParserJobClient struct {
	job parserv1.ParserJob
}

func (client fakeParserJobClient) StartAsync(_ context.Context, _ parserv1.SearchQuery) (parserv1.ParserJob, error) {
	return client.job, nil
}

func (client fakeParserJobClient) Get(_ context.Context, id string) (parserv1.ParserJob, bool, error) {
	if client.job.ID == id {
		return client.job, true, nil
	}
	return parserv1.ParserJob{}, false, nil
}

func TestSearchJobsHandlerStartsParserJob(t *testing.T) {
	handler := NewSearchJobsHandler(fakeParserJobClient{job: parserv1.ParserJob{
		ID:     "parser-job-1",
		Status: parserv1.ParserJobStatusQueued,
	}})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/search/jobs", bytes.NewBufferString(`{"query":"go kafka","sources":["habr"],"limit":5}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", response.Code)
	}
	var payload SearchJobResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Job.ID != "parser-job-1" {
		t.Fatalf("unexpected job id: %s", payload.Job.ID)
	}
}

func TestSearchJobsHandlerGetsParserJob(t *testing.T) {
	handler := NewSearchJobsHandler(fakeParserJobClient{job: parserv1.ParserJob{
		ID:              "parser-job-1",
		Status:          parserv1.ParserJobStatusCompleted,
		CandidatesCount: 1,
		Candidates: []parserv1.ArticleCandidate{
			{SourceName: "vc", ExternalID: "2", Title: "Travel"},
		},
	}})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/search/jobs/parser-job-1", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload SearchJobResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Job.Status != parserv1.ParserJobStatusCompleted {
		t.Fatalf("unexpected status: %s", payload.Job.Status)
	}
	if len(payload.Job.Candidates) != 1 {
		t.Fatalf("expected candidates in gateway response, got %d", len(payload.Job.Candidates))
	}
}
