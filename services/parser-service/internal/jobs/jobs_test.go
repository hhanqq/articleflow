package jobs

import (
	"context"
	"errors"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type fakeSearchPublisher struct {
	candidates []parserv1.ArticleCandidate
	err        error
}

func (publisher fakeSearchPublisher) SearchAndPublish(_ context.Context, _ parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	if publisher.err != nil {
		return nil, publisher.err
	}
	return publisher.candidates, nil
}

func TestRunSearchJobCompletesAndStoresCandidateCount(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, fakeSearchPublisher{
		candidates: []parserv1.ArticleCandidate{
			{Title: "First"},
			{Title: "Second"},
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
	stored, ok := store.FindByID(job.ID)
	if !ok {
		t.Fatal("expected stored job")
	}
	if stored.Status != parserv1.ParserJobStatusCompleted {
		t.Fatalf("expected stored completed status, got %s", stored.Status)
	}
}

func TestRunSearchJobStoresFailure(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, fakeSearchPublisher{err: errors.New("habr unavailable")})
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
