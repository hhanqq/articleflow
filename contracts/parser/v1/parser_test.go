package parserv1

import "testing"

func TestSearchQueryValidate(t *testing.T) {
	query := SearchQuery{
		Text:     "go kafka microservices",
		Sources:  []string{"habr"},
		Limit:    20,
		Language: "ru",
	}

	if err := query.Validate(); err != nil {
		t.Fatalf("expected valid query, got %v", err)
	}
}

func TestSearchQueryValidateRequiresText(t *testing.T) {
	query := SearchQuery{Limit: 20}

	if err := query.Validate(); err == nil {
		t.Fatal("expected validation error for empty query")
	}
}

func TestSearchQueryNormalizeDefaultsLimit(t *testing.T) {
	query := SearchQuery{Text: "go"}

	normalized := query.Normalize()

	if normalized.Limit != 20 {
		t.Fatalf("expected default limit 20, got %d", normalized.Limit)
	}
	if normalized.Language != "ru" {
		t.Fatalf("expected default language ru, got %s", normalized.Language)
	}
}

func TestArticleCandidateCarriesFullContent(t *testing.T) {
	candidate := ArticleCandidate{Content: "Full parsed article content"}

	if candidate.Content != "Full parsed article content" {
		t.Fatalf("unexpected content: %s", candidate.Content)
	}
}

func TestParserJobStatuses(t *testing.T) {
	if ParserJobStatusQueued != "queued" {
		t.Fatalf("unexpected queued status: %s", ParserJobStatusQueued)
	}
	if ParserJobStatusRunning != "running" {
		t.Fatalf("unexpected running status: %s", ParserJobStatusRunning)
	}
	if ParserJobStatusCompleted != "completed" {
		t.Fatalf("unexpected completed status: %s", ParserJobStatusCompleted)
	}
	if ParserJobStatusFailed != "failed" {
		t.Fatalf("unexpected failed status: %s", ParserJobStatusFailed)
	}
}

func TestParserJobCarriesExecutionResult(t *testing.T) {
	job := ParserJob{
		ID:              "job-1",
		Status:          ParserJobStatusCompleted,
		CandidatesCount: 3,
		Candidates: []ArticleCandidate{
			{SourceName: "habr", ExternalID: "1", Title: "First"},
		},
		Error: "",
	}

	if job.CandidatesCount != 3 {
		t.Fatalf("unexpected candidates count: %d", job.CandidatesCount)
	}
	if len(job.Candidates) != 1 {
		t.Fatalf("expected job candidates to be carried, got %d", len(job.Candidates))
	}
}
