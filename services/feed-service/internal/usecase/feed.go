package usecase

import (
	"sort"
	"sync"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
)

type MemoryFeed struct {
	mu       sync.RWMutex
	articles []articlev1.ArticlePreview
}

func NewMemoryFeed() *MemoryFeed {
	return &MemoryFeed{}
}

func (feed *MemoryFeed) AddArticle(article articlev1.ArticlePreview) {
	feed.mu.Lock()
	defer feed.mu.Unlock()
	feed.articles = append(feed.articles, article)
}

func (feed *MemoryFeed) List(limit int) []feedv1.FeedItem {
	feed.mu.RLock()
	defer feed.mu.RUnlock()

	articles := append([]articlev1.ArticlePreview(nil), feed.articles...)
	sort.SliceStable(articles, func(i, j int) bool {
		return articles[i].PublishedAt.After(articles[j].PublishedAt)
	})
	if limit <= 0 || limit > len(articles) {
		limit = len(articles)
	}

	items := make([]feedv1.FeedItem, 0, limit)
	for _, article := range articles[:limit] {
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
	return items
}

func scoreByFreshness(publishedAt time.Time) float64 {
	if publishedAt.IsZero() {
		return 0
	}
	return float64(publishedAt.Unix())
}

