package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
)

func TestIngestDiscoveredStoresArticleAndReturnsCreatedEvent(t *testing.T) {
	store := NewMemoryArticleStore()
	usecase := NewIngestUsecase(store)
	discovered := eventsv1.ArticleDiscoveredEvent{
		SourceName:   "habr",
		ExternalID:   "habr-123",
		URL:          "https://habr.com/ru/articles/123/",
		Title:        "Go microservices",
		Summary:      "Short article summary",
		Content:      "Full parsed article content from Habr",
		Tags:         []string{"go", "microservices"},
		PublishedAt:  time.Date(2026, 7, 7, 10, 30, 0, 0, time.UTC),
		DiscoveredAt: time.Now().UTC(),
	}

	created, err := usecase.IngestDiscovered(context.Background(), discovered)

	if err != nil {
		t.Fatalf("expected no ingest error, got %v", err)
	}
	if created.ArticleID == "" {
		t.Fatal("expected created article id")
	}
	if created.ArticleID != "habr:habr-123" {
		t.Fatalf("expected feed-aligned article id, got %s", created.ArticleID)
	}
	if created.URL != discovered.URL {
		t.Fatalf("expected url %s, got %s", discovered.URL, created.URL)
	}
	article, ok := store.FindByURL(discovered.URL)
	if !ok {
		t.Fatal("expected article stored by url")
	}
	if article.Title != discovered.Title {
		t.Fatalf("unexpected stored title: %s", article.Title)
	}
	if article.Content != discovered.Content {
		t.Fatalf("unexpected stored content: %s", article.Content)
	}
}

func TestIngestDiscoveredDeduplicatesByURL(t *testing.T) {
	store := NewMemoryArticleStore()
	usecase := NewIngestUsecase(store)
	discovered := eventsv1.ArticleDiscoveredEvent{
		SourceName:   "habr",
		ExternalID:   "habr-123",
		URL:          "https://habr.com/ru/articles/123/",
		Title:        "Go microservices",
		DiscoveredAt: time.Now().UTC(),
	}

	first, err := usecase.IngestDiscovered(context.Background(), discovered)
	if err != nil {
		t.Fatalf("first ingest failed: %v", err)
	}
	second, err := usecase.IngestDiscovered(context.Background(), discovered)
	if err != nil {
		t.Fatalf("second ingest failed: %v", err)
	}

	if first.ArticleID != second.ArticleID {
		t.Fatalf("expected duplicate to return same id, got %s and %s", first.ArticleID, second.ArticleID)
	}
}

type failingArticleStore struct{}

func (failingArticleStore) Save(_ context.Context, _ articlev1.Article) (articlev1.Article, error) {
	return articlev1.Article{}, errors.New("storage unavailable")
}

func (failingArticleStore) GetByID(_ context.Context, _ string) (articlev1.Article, bool, error) {
	return articlev1.Article{}, false, errors.New("storage unavailable")
}

func TestIngestDiscoveredReturnsStoreError(t *testing.T) {
	usecase := NewIngestUsecase(failingArticleStore{})
	discovered := eventsv1.ArticleDiscoveredEvent{
		SourceName:   "habr",
		ExternalID:   "habr-123",
		URL:          "https://habr.com/ru/articles/123/",
		Title:        "Go microservices",
		DiscoveredAt: time.Now().UTC(),
	}

	_, err := usecase.IngestDiscovered(context.Background(), discovered)

	if err == nil {
		t.Fatal("expected storage error")
	}
}
