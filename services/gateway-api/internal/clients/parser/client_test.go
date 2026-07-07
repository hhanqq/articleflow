package parserclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

func TestClientStartsParserJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", request.Method)
		}
		if request.URL.Path != "/api/v1/parser/jobs" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		_ = json.NewEncoder(response).Encode(jobResponse{Job: parserv1.ParserJob{
			ID:     "parser-job-1",
			Status: parserv1.ParserJobStatusQueued,
		}})
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	job, err := client.StartAsync(context.Background(), parserv1.SearchQuery{Text: "go kafka", Limit: 5})

	if err != nil {
		t.Fatalf("start job: %v", err)
	}
	if job.ID != "parser-job-1" {
		t.Fatalf("unexpected job id: %s", job.ID)
	}
}

func TestClientGetsParserJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", request.Method)
		}
		if request.URL.Path != "/api/v1/parser/jobs/parser-job-1" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		_ = json.NewEncoder(response).Encode(jobResponse{Job: parserv1.ParserJob{
			ID:     "parser-job-1",
			Status: parserv1.ParserJobStatusCompleted,
		}})
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	job, ok, err := client.Get(context.Background(), "parser-job-1")

	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if !ok {
		t.Fatal("expected job found")
	}
	if job.Status != parserv1.ParserJobStatusCompleted {
		t.Fatalf("unexpected status: %s", job.Status)
	}
}
