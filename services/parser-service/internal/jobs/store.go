package jobs

import (
	"context"
	"sort"
	"sync"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type Store interface {
	Save(ctx context.Context, job parserv1.ParserJob) (parserv1.ParserJob, error)
	FindByID(ctx context.Context, id string) (parserv1.ParserJob, bool, error)
	List(ctx context.Context, limit int) ([]parserv1.ParserJob, error)
}

type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]parserv1.ParserJob
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{jobs: make(map[string]parserv1.ParserJob)}
}

func (store *MemoryStore) Save(_ context.Context, job parserv1.ParserJob) (parserv1.ParserJob, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.jobs[job.ID] = job
	return job, nil
}

func (store *MemoryStore) FindByID(_ context.Context, id string) (parserv1.ParserJob, bool, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	job, ok := store.jobs[id]
	return job, ok, nil
}

func (store *MemoryStore) List(_ context.Context, limit int) ([]parserv1.ParserJob, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	jobs := make([]parserv1.ParserJob, 0, len(store.jobs))
	for _, job := range store.jobs {
		jobs = append(jobs, job)
	}
	sort.SliceStable(jobs, func(left, right int) bool {
		return jobs[left].UpdatedAt.After(jobs[right].UpdatedAt)
	})
	return jobs[:min(limit, len(jobs))], nil
}
