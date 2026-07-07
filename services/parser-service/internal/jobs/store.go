package jobs

import (
	"sync"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type Store interface {
	Save(job parserv1.ParserJob) parserv1.ParserJob
	FindByID(id string) (parserv1.ParserJob, bool)
}

type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]parserv1.ParserJob
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{jobs: make(map[string]parserv1.ParserJob)}
}

func (store *MemoryStore) Save(job parserv1.ParserJob) parserv1.ParserJob {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.jobs[job.ID] = job
	return job
}

func (store *MemoryStore) FindByID(id string) (parserv1.ParserJob, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	job, ok := store.jobs[id]
	return job, ok
}
