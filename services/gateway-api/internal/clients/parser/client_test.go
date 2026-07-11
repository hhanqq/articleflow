package parserclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

func TestClientStartsParserJob(t *testing.T) {
	fromDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", request.Method)
		}
		if request.URL.Path != "/api/v1/parser/jobs" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		var payload jobRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.FromDate == nil || !payload.FromDate.Equal(fromDate) {
			t.Fatalf("expected from date to be forwarded, got %#v", payload.FromDate)
		}
		_ = json.NewEncoder(response).Encode(jobResponse{Job: parserv1.ParserJob{
			ID:     "parser-job-1",
			Status: parserv1.ParserJobStatusQueued,
		}})
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	job, err := client.StartAsync(context.Background(), parserv1.SearchQuery{Text: "go kafka", Limit: 5, FromDate: &fromDate})

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
			ID:              "parser-job-1",
			Status:          parserv1.ParserJobStatusCompleted,
			CandidatesCount: 1,
			Candidates: []parserv1.ArticleCandidate{
				{SourceName: "vc", ExternalID: "2", Title: "Travel"},
			},
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
	if len(job.Candidates) != 1 {
		t.Fatalf("expected candidates from parser-service, got %d", len(job.Candidates))
	}
}

func TestClientListsParserJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/parser/jobs" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("limit") != "5" {
			t.Fatalf("unexpected limit query: %s", request.URL.RawQuery)
		}
		_ = json.NewEncoder(response).Encode(jobsResponse{Jobs: []parserv1.ParserJob{
			{ID: "parser-job-2", Status: parserv1.ParserJobStatusCompleted},
			{ID: "parser-job-1", Status: parserv1.ParserJobStatusFailed},
		}})
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	jobs, err := client.List(context.Background(), 5)

	if err != nil {
		t.Fatalf("list parser jobs: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	if jobs[0].ID != "parser-job-2" {
		t.Fatalf("unexpected first job id: %s", jobs[0].ID)
	}
}

func TestClientListsParserSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/parser/sources" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		_ = json.NewEncoder(response).Encode(sourcesResponse{Sources: []parserv1.ParserSource{
			{Name: "habr", DisplayName: "Habr", Kind: "html_rss", Enabled: true, Searchable: true},
		}})
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	sources, err := client.ListSources(context.Background())

	if err != nil {
		t.Fatalf("list parser sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Name != "habr" {
		t.Fatalf("unexpected source: %#v", sources[0])
	}
}

func TestClientUpdatesParserSourceEnabledState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", request.Method)
		}
		if request.URL.Path != "/api/v1/parser/sources/vc" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		body, _ := io.ReadAll(request.Body)
		if !bytes.Contains(body, []byte(`"enabled":false`)) {
			t.Fatalf("expected enabled=false body, got %s", string(body))
		}
		_ = json.NewEncoder(response).Encode(sourceResponse{Source: parserv1.ParserSource{
			Name: "vc", DisplayName: "vc.ru", Enabled: false, Searchable: true,
		}})
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	source, ok, err := client.SetSourceEnabled(context.Background(), "vc", false)

	if err != nil {
		t.Fatalf("update source: %v", err)
	}
	if !ok {
		t.Fatal("expected source found")
	}
	if source.Name != "vc" || source.Enabled {
		t.Fatalf("unexpected source: %#v", source)
	}
}
