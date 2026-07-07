package usecase

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sync"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
)

type MemoryArticleStore struct {
	mu    sync.RWMutex
	byURL map[string]articlev1.Article
	byID  map[string]articlev1.Article
}

type ArticleStore interface {
	Save(ctx context.Context, article articlev1.Article) (articlev1.Article, error)
	GetByID(ctx context.Context, id string) (articlev1.Article, bool, error)
}

func NewMemoryArticleStore() *MemoryArticleStore {
	return &MemoryArticleStore{
		byURL: make(map[string]articlev1.Article),
		byID:  make(map[string]articlev1.Article),
	}
}

func (store *MemoryArticleStore) Save(_ context.Context, article articlev1.Article) (articlev1.Article, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if existing, ok := store.byURL[article.URL]; ok {
		return existing, nil
	}
	store.byURL[article.URL] = article
	store.byID[article.ID] = article
	return article, nil
}

func (store *MemoryArticleStore) FindByURL(url string) (articlev1.Article, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	article, ok := store.byURL[url]
	return article, ok
}

func (store *MemoryArticleStore) GetByID(_ context.Context, id string) (articlev1.Article, bool, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	article, ok := store.byID[id]
	return article, ok, nil
}

type IngestUsecase struct {
	store ArticleStore
}

func NewIngestUsecase(store ArticleStore) *IngestUsecase {
	return &IngestUsecase{store: store}
}

func (usecase *IngestUsecase) IngestDiscovered(ctx context.Context, event eventsv1.ArticleDiscoveredEvent) (eventsv1.ArticleCreatedEvent, error) {
	if err := event.Validate(); err != nil {
		return eventsv1.ArticleCreatedEvent{}, err
	}
	select {
	case <-ctx.Done():
		return eventsv1.ArticleCreatedEvent{}, ctx.Err()
	default:
	}

	article := articlev1.Article{
		ID:          articleID(event),
		SourceName:  event.SourceName,
		ExternalID:  event.ExternalID,
		URL:         event.URL,
		Title:       event.Title,
		Summary:     event.Summary,
		Content:     event.Content,
		Author:      event.Author,
		Tags:        event.Tags,
		Language:    event.Language,
		PublishedAt: event.PublishedAt,
		ParsedAt:    time.Now().UTC(),
	}
	article, err := usecase.store.Save(ctx, article)
	if err != nil {
		return eventsv1.ArticleCreatedEvent{}, err
	}

	return eventsv1.ArticleCreatedEvent{
		ArticleID:   article.ID,
		SourceName:  article.SourceName,
		ExternalID:  article.ExternalID,
		URL:         article.URL,
		Title:       article.Title,
		CreatedAt:   article.ParsedAt,
		PublishedAt: article.PublishedAt,
	}, nil
}

func articleID(event eventsv1.ArticleDiscoveredEvent) string {
	if event.SourceName != "" && event.ExternalID != "" {
		return event.SourceName + ":" + event.ExternalID
	}
	return stableArticleID(event.URL)
}

func stableArticleID(value string) string {
	sum := sha1.Sum([]byte(value))
	return "article-" + hex.EncodeToString(sum[:8])
}
