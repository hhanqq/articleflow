package jobs

import (
	"context"
	"errors"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type fakeSearchPublisher struct {
	result parserv1.SearchResult
	err    error
}

func (publisher fakeSearchPublisher) SearchAndPublish(_ context.Context, _ parserv1.SearchQuery) (parserv1.SearchResult, error) {
	if publisher.err != nil {
		return publisher.result, publisher.err
	}
	return publisher.result, nil
}

func TestRunSearchJobCompletesAndStoresCandidates(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, fakeSearchPublisher{
		result: parserv1.SearchResult{
			Candidates: []parserv1.ArticleCandidate{
				{SourceName: "habr", ExternalID: "1", Title: "First"},
				{SourceName: "vc", ExternalID: "2", Title: "Second"},
			},
			SourceStats: []parserv1.SourceStats{
				{SourceName: "habr", Status: parserv1.SourceStatusOK, FoundCount: 3, AcceptedCount: 2, ReturnedCount: 1},
				{SourceName: "vc", Status: parserv1.SourceStatusOK, FoundCount: 2, AcceptedCount: 2, ReturnedCount: 1},
			},
		},
	})

	job, err := manager.Start(context.Background(), parserv1.SearchQuery{Text: "go kafka", Limit: 10})
	if err != nil {
		t.Fatalf("start job: %v", err)
	}
	completed, err := manager.Run(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("run job: %v", err)
	}

	if completed.Status != parserv1.ParserJobStatusCompleted {
		t.Fatalf("expected completed status, got %s", completed.Status)
	}
	if completed.CandidatesCount != 2 {
		t.Fatalf("expected 2 candidates, got %d", completed.CandidatesCount)
	}
	if len(completed.Candidates) != 2 {
		t.Fatalf("expected 2 stored candidates, got %d", len(completed.Candidates))
	}
	if len(completed.SourceStats) != 2 {
		t.Fatalf("expected stored source stats, got %d", len(completed.SourceStats))
	}
	if completed.Candidates[0].Title != "First" {
		t.Fatalf("unexpected first candidate: %s", completed.Candidates[0].Title)
	}
	stored, ok := store.FindByID(job.ID)
	if !ok {
		t.Fatal("expected stored job")
	}
	if stored.Status != parserv1.ParserJobStatusCompleted {
		t.Fatalf("expected stored completed status, got %s", stored.Status)
	}
	if len(stored.Candidates) != 2 {
		t.Fatalf("expected stored candidates, got %d", len(stored.Candidates))
	}
	if len(stored.SourceStats) != 2 {
		t.Fatalf("expected stored source stats, got %d", len(stored.SourceStats))
	}
}

func TestRunSearchJobStoresFailure(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, fakeSearchPublisher{
		result: parserv1.SearchResult{
			SourceStats: []parserv1.SourceStats{
				{SourceName: "habr", Status: parserv1.SourceStatusFailed, Error: "habr unavailable"},
			},
		},
		err: errors.New("habr unavailable"),
	})
	job, err := manager.Start(context.Background(), parserv1.SearchQuery{Text: "go kafka", Limit: 10})
	if err != nil {
		t.Fatalf("start job: %v", err)
	}

	failed, err := manager.Run(context.Background(), job.ID)

	if err == nil {
		t.Fatal("expected run error")
	}
	if failed.Status != parserv1.ParserJobStatusFailed {
		t.Fatalf("expected failed status, got %s", failed.Status)
	}
	if failed.Error != "habr unavailable" {
		t.Fatalf("unexpected error: %s", failed.Error)
	}
	if len(failed.SourceStats) != 1 {
		t.Fatalf("expected failed source stats, got %d", len(failed.SourceStats))
	}
	if failed.SourceStats[0].Status != parserv1.SourceStatusFailed {
		t.Fatalf("unexpected source status: %s", failed.SourceStats[0].Status)
	}
}

func TestStartAsyncReturnsQueuedJob(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, fakeSearchPublisher{})

	job, err := manager.StartAsync(context.Background(), parserv1.SearchQuery{Text: "go kafka", Limit: 10})

	if err != nil {
		t.Fatalf("start async: %v", err)
	}
	if job.Status != parserv1.ParserJobStatusQueued {
		t.Fatalf("expected queued status, got %s", job.Status)
	}
}
