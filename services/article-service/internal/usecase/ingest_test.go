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

func TestMemoryArticleStoreSearchesStoredArticles(t *testing.T) {
	store := NewMemoryArticleStore()
	articles := []articlev1.Article{
		{
			ID:          "vc:1",
			SourceName:  "vc",
			ExternalID:  "1",
			URL:         "https://vc.ru/travel/1",
			Title:       "Путешествие в Китай",
			Summary:     "Маршрут по Пекину",
			PublishedAt: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:          "habr:2",
			SourceName:  "habr",
			ExternalID:  "2",
			URL:         "https://habr.com/ru/articles/2/",
			Title:       "Как планировать путешествие в Китай",
			Summary:     "Опыт поездки и бюджет",
			PublishedAt: time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:          "habr:3",
			SourceName:  "habr",
			ExternalID:  "3",
			URL:         "https://habr.com/ru/articles/3/",
			Title:       "Go Kafka",
			PublishedAt: time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC),
		},
	}
	for _, article := range articles {
		if _, err := store.Save(context.Background(), article); err != nil {
			t.Fatalf("save article: %v", err)
		}
	}

	found, err := store.Search(context.Background(), articlev1.SearchQuery{Text: "путешествие в китай", Limit: 10})

	if err != nil {
		t.Fatalf("search articles: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("expected 2 articles, got %d", len(found))
	}
	if found[0].ID != "habr:2" {
		t.Fatalf("expected newest match first, got %s", found[0].ID)
	}
}

func TestMemoryArticleStoreSearchAppliesFiltersAndOffset(t *testing.T) {
	store := NewMemoryArticleStore()
	articles := []articlev1.Article{
		{
			ID:          "vc:1",
			SourceName:  "vc",
			URL:         "https://vc.ru/travel/1",
			Title:       "Путешествие в Китай",
			Tags:        []string{"Travel *"},
			PublishedAt: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:          "vc:2",
			SourceName:  "vc",
			URL:         "https://vc.ru/travel/2",
			Title:       "Путешествие в Китай",
			Tags:        []string{"Travel *"},
			PublishedAt: time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:          "habr:3",
			SourceName:  "habr",
			URL:         "https://habr.com/ru/articles/3/",
			Title:       "Путешествие в Китай",
			Tags:        []string{"backend"},
			PublishedAt: time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC),
		},
	}
	for _, article := range articles {
		if _, err := store.Save(context.Background(), article); err != nil {
			t.Fatalf("save article: %v", err)
		}
	}
	fromDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	found, err := store.Search(context.Background(), articlev1.SearchQuery{
		Text:     "путешествие китай",
		Sources:  []string{"vc"},
		Tags:     []string{"travel"},
		FromDate: &fromDate,
		Limit:    1,
		Offset:   1,
	})

	if err != nil {
		t.Fatalf("search articles: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 article, got %d", len(found))
	}
	if found[0].ID != "vc:1" {
		t.Fatalf("unexpected article: %s", found[0].ID)
	}
}

type failingArticleStore struct{}

func (failingArticleStore) Save(_ context.Context, _ articlev1.Article) (articlev1.Article, error) {
	return articlev1.Article{}, errors.New("storage unavailable")
}

func (failingArticleStore) GetByID(_ context.Context, _ string) (articlev1.Article, bool, error) {
	return articlev1.Article{}, false, errors.New("storage unavailable")
}

func (failingArticleStore) Search(_ context.Context, _ articlev1.SearchQuery) ([]articlev1.Article, error) {
	return nil, errors.New("storage unavailable")
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
