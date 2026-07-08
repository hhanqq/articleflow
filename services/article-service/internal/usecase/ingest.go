package usecase

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
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
	Search(ctx context.Context, query articlev1.SearchQuery) ([]articlev1.Article, error)
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

func (store *MemoryArticleStore) Search(_ context.Context, query articlev1.SearchQuery) ([]articlev1.Article, error) {
	query = query.Normalize()
	terms := searchTerms(query.Text)
	sources := sourceSet(query.Sources)
	tags := tagSet(query.Tags)

	store.mu.RLock()
	defer store.mu.RUnlock()
	articles := make([]articlev1.Article, 0, len(store.byID))
	for _, article := range store.byID {
		if len(sources) > 0 && !sources[strings.ToLower(article.SourceName)] {
			continue
		}
		if len(tags) > 0 && !articleHasAnyTag(article, tags) {
			continue
		}
		if query.FromDate != nil && article.PublishedAt.Before(*query.FromDate) {
			continue
		}
		if query.ToDate != nil && article.PublishedAt.After(*query.ToDate) {
			continue
		}
		text := strings.ToLower(strings.Join([]string{
			article.Title,
			article.Summary,
			article.Content,
			article.URL,
			article.Author,
			strings.Join(article.Tags, " "),
		}, " "))
		if !matchesAllTerms(text, terms) {
			continue
		}
		articles = append(articles, article)
	}
	sort.SliceStable(articles, func(i, j int) bool {
		return articles[i].PublishedAt.After(articles[j].PublishedAt)
	})
	articles = interleaveArticlesBySource(articles, query.Sources)
	if query.Offset >= len(articles) {
		return []articlev1.Article{}, nil
	}
	articles = articles[query.Offset:]
	if len(articles) > query.Limit {
		articles = articles[:query.Limit]
	}
	return articles, nil
}

func interleaveArticlesBySource(articles []articlev1.Article, sourceOrder []string) []articlev1.Article {
	if len(articles) == 0 {
		return nil
	}
	order := cleanSourceOrder(sourceOrder)
	known := make(map[string]bool, len(order))
	for _, source := range order {
		known[source] = true
	}
	buckets := make(map[string][]articlev1.Article)
	for _, article := range articles {
		source := strings.ToLower(strings.TrimSpace(article.SourceName))
		if source == "" {
			source = "_unknown"
		}
		if !known[source] {
			order = append(order, source)
			known[source] = true
		}
		buckets[source] = append(buckets[source], article)
	}
	result := make([]articlev1.Article, 0, len(articles))
	for index := 0; len(result) < len(articles); index++ {
		added := false
		for _, source := range order {
			bucket := buckets[source]
			if index >= len(bucket) {
				continue
			}
			result = append(result, bucket[index])
			added = true
		}
		if !added {
			break
		}
	}
	return result
}

func cleanSourceOrder(sources []string) []string {
	order := make([]string, 0, len(sources))
	seen := make(map[string]bool, len(sources))
	for _, source := range sources {
		source = strings.ToLower(strings.TrimSpace(source))
		if source == "" || seen[source] {
			continue
		}
		seen[source] = true
		order = append(order, source)
	}
	return order
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

func searchTerms(text string) []string {
	fields := strings.Fields(strings.ToLower(text))
	terms := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.Trim(field, `"'.,:;!?()[]{}<>`)
		if len([]rune(field)) < 2 || isSearchStopWord(field) {
			continue
		}
		terms = append(terms, field)
	}
	return terms
}

func matchesAllTerms(text string, terms []string) bool {
	if len(terms) == 0 {
		return false
	}
	for _, term := range terms {
		if !strings.Contains(text, term) {
			return false
		}
	}
	return true
}

func sourceSet(sources []string) map[string]bool {
	set := make(map[string]bool, len(sources))
	for _, source := range sources {
		source = strings.ToLower(strings.TrimSpace(source))
		if source != "" {
			set[source] = true
		}
	}
	return set
}

func articleHasAnyTag(article articlev1.Article, tags map[string]bool) bool {
	for _, tag := range article.Tags {
		if tags[normalizeTag(tag)] {
			return true
		}
	}
	return false
}

func tagSet(tags []string) map[string]bool {
	set := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tag = normalizeTag(tag)
		if tag != "" {
			set[tag] = true
		}
	}
	return set
}

func normalizeTag(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	tag = strings.TrimSpace(strings.TrimSuffix(tag, "*"))
	return tag
}

func isSearchStopWord(word string) bool {
	switch word {
	case "и", "в", "во", "на", "по", "с", "со", "о", "об", "от", "до", "для", "из", "за", "к", "ко", "a", "an", "the", "of", "to", "in", "on", "for", "and":
		return true
	default:
		return false
	}
}
