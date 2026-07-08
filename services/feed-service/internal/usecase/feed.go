package usecase

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
)

type FeedStore interface {
	UpsertScoredItem(event eventsv1.FeedItemScoredEvent) error
	List(limit int) ([]feedv1.FeedItem, error)
}

type MemoryFeed struct {
	mu       sync.RWMutex
	articles []articlev1.ArticlePreview
	scored   map[string]feedv1.FeedItem
}

func NewMemoryFeed() *MemoryFeed {
	return &MemoryFeed{scored: make(map[string]feedv1.FeedItem)}
}

func (feed *MemoryFeed) AddArticle(article articlev1.ArticlePreview) {
	feed.mu.Lock()
	defer feed.mu.Unlock()
	feed.articles = append(feed.articles, article)
}

func (feed *MemoryFeed) UpsertScoredItem(event eventsv1.FeedItemScoredEvent) error {
	if event.ArticleID == "" {
		return errors.New("article id is required")
	}
	feed.mu.Lock()
	defer feed.mu.Unlock()
	feed.scored[event.ArticleID] = feedv1.FeedItem{
		ArticleID:    event.ArticleID,
		Title:        event.Title,
		Summary:      event.Summary,
		SourceName:   event.SourceName,
		URL:          event.URL,
		Tags:         append([]string(nil), event.Tags...),
		Score:        event.Score,
		ScoreReasons: append([]string(nil), event.ScoreReasons...),
		PublishedAt:  event.PublishedAt,
	}
	return nil
}

func (feed *MemoryFeed) List(limit int) ([]feedv1.FeedItem, error) {
	feed.mu.RLock()
	defer feed.mu.RUnlock()

	if len(feed.scored) > 0 {
		return feed.listScored(limit), nil
	}

	articles := append([]articlev1.ArticlePreview(nil), feed.articles...)
	sort.SliceStable(articles, func(i, j int) bool {
		return articles[i].PublishedAt.After(articles[j].PublishedAt)
	})
	if limit <= 0 || limit > len(articles) {
		limit = len(articles)
	}

	items := make([]feedv1.FeedItem, 0, len(articles))
	for _, article := range articles {
		items = append(items, feedv1.FeedItem{
			ArticleID:   article.ID,
			Title:       article.Title,
			Summary:     article.Summary,
			SourceName:  article.SourceName,
			URL:         article.URL,
			Tags:        article.Tags,
			Score:       scoreByFreshness(article.PublishedAt),
			PublishedAt: article.PublishedAt,
		})
	}
	return limitFeedItems(interleaveFeedItemsBySource(items), limit), nil
}

func (feed *MemoryFeed) listScored(limit int) []feedv1.FeedItem {
	items := make([]feedv1.FeedItem, 0, len(feed.scored))
	for _, item := range feed.scored {
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score == items[j].Score {
			return items[i].PublishedAt.After(items[j].PublishedAt)
		}
		return items[i].Score > items[j].Score
	})
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	return limitFeedItems(interleaveFeedItemsBySource(items), limit)
}

func interleaveFeedItemsBySource(items []feedv1.FeedItem) []feedv1.FeedItem {
	if len(items) == 0 {
		return nil
	}
	order := make([]string, 0)
	seenSources := make(map[string]bool)
	buckets := make(map[string][]feedv1.FeedItem)
	for _, item := range items {
		source := strings.ToLower(strings.TrimSpace(item.SourceName))
		if source == "" {
			source = "_unknown"
		}
		if !seenSources[source] {
			seenSources[source] = true
			order = append(order, source)
		}
		buckets[source] = append(buckets[source], item)
	}
	result := make([]feedv1.FeedItem, 0, len(items))
	for index := 0; len(result) < len(items); index++ {
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

func limitFeedItems(items []feedv1.FeedItem, limit int) []feedv1.FeedItem {
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	return append([]feedv1.FeedItem(nil), items[:limit]...)
}

func scoreByFreshness(publishedAt time.Time) float64 {
	if publishedAt.IsZero() {
		return 0
	}
	return float64(publishedAt.Unix())
}
