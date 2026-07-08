package jobs

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type SearchPublisher interface {
	SearchAndPublish(ctx context.Context, query parserv1.SearchQuery) (parserv1.SearchResult, error)
}

type Manager struct {
	store     Store
	publisher SearchPublisher
}

func NewManager(store Store, publisher SearchPublisher) *Manager {
	return &Manager{store: store, publisher: publisher}
}

func (manager *Manager) Start(ctx context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error) {
	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return parserv1.ParserJob{}, err
	}
	select {
	case <-ctx.Done():
		return parserv1.ParserJob{}, ctx.Err()
	default:
	}
	now := time.Now().UTC()
	job := parserv1.ParserJob{
		ID:        stableJobID(query, now),
		Query:     query,
		Status:    parserv1.ParserJobStatusQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return manager.store.Save(ctx, job)
}

func (manager *Manager) StartAsync(ctx context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error) {
	job, err := manager.Start(ctx, query)
	if err != nil {
		return parserv1.ParserJob{}, err
	}
	go func() {
		_, _ = manager.Run(context.Background(), job.ID)
	}()
	return job, nil
}

func (manager *Manager) Run(ctx context.Context, id string) (parserv1.ParserJob, error) {
	job, ok, err := manager.store.FindByID(ctx, id)
	if err != nil {
		return parserv1.ParserJob{}, err
	}
	if !ok {
		return parserv1.ParserJob{}, errors.New("parser job not found")
	}
	job.Status = parserv1.ParserJobStatusRunning
	job.UpdatedAt = time.Now().UTC()
	if _, err := manager.store.Save(ctx, job); err != nil {
		return parserv1.ParserJob{}, err
	}

	result, err := manager.publisher.SearchAndPublish(ctx, job.Query)
	job.UpdatedAt = time.Now().UTC()
	job.SourceStats = append([]parserv1.SourceStats(nil), result.SourceStats...)
	if err != nil {
		job.Status = parserv1.ParserJobStatusFailed
		job.Error = err.Error()
		_, saveErr := manager.store.Save(ctx, job)
		if saveErr != nil {
			return job, saveErr
		}
		return job, err
	}
	job.Status = parserv1.ParserJobStatusCompleted
	job.CandidatesCount = len(result.Candidates)
	job.Candidates = append([]parserv1.ArticleCandidate(nil), result.Candidates...)
	job.Error = ""
	return manager.store.Save(ctx, job)
}

func (manager *Manager) Get(ctx context.Context, id string) (parserv1.ParserJob, bool, error) {
	return manager.store.FindByID(ctx, id)
}

func (manager *Manager) List(ctx context.Context, limit int) ([]parserv1.ParserJob, error) {
	return manager.store.List(ctx, limit)
}

func stableJobID(query parserv1.SearchQuery, createdAt time.Time) string {
	sum := sha1.Sum([]byte(query.Text + createdAt.Format(time.RFC3339Nano)))
	return "parser-job-" + hex.EncodeToString(sum[:8])
}
