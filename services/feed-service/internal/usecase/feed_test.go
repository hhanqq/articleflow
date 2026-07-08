package usecase

import (
	"testing"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
)

func TestListReturnsLatestArticlesAsFeedItems(t *testing.T) {
	feed := NewMemoryFeed()
	oldArticle := articlev1.ArticlePreview{
		ID:          "old",
		Title:       "Old article",
		SourceName:  "habr",
		URL:         "https://habr.com/old",
		PublishedAt: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
	}
	newArticle := articlev1.ArticlePreview{
		ID:          "new",
		Title:       "New article",
		SourceName:  "habr",
		URL:         "https://habr.com/new",
		PublishedAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
	}
	feed.AddArticle(oldArticle)
	feed.AddArticle(newArticle)

	items, err := feed.List(10)
	if err != nil {
		t.Fatalf("list feed: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ArticleID != "new" {
		t.Fatalf("expected newest article first, got %s", items[0].ArticleID)
	}
	if items[0].Score <= items[1].Score {
		t.Fatalf("expected newer article score to be higher")
	}
}

func TestListAppliesLimit(t *testing.T) {
	feed := NewMemoryFeed()
	feed.AddArticle(articlev1.ArticlePreview{ID: "1", Title: "One", PublishedAt: time.Now().UTC()})
	feed.AddArticle(articlev1.ArticlePreview{ID: "2", Title: "Two", PublishedAt: time.Now().UTC()})

	items, err := feed.List(1)
	if err != nil {
		t.Fatalf("list feed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestUpsertScoredItemSortsByRankingScore(t *testing.T) {
	feed := NewMemoryFeed()
	if err := feed.UpsertScoredItem(eventsv1.FeedItemScoredEvent{
		ArticleID:   "low",
		Title:       "Low score",
		SourceName:  "habr",
		URL:         "https://habr.com/low",
		Score:       10,
		PublishedAt: time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("upsert low item: %v", err)
	}
	if err := feed.UpsertScoredItem(eventsv1.FeedItemScoredEvent{
		ArticleID:   "high",
		Title:       "High score",
		SourceName:  "habr",
		URL:         "https://habr.com/high",
		Score:       99,
		PublishedAt: time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("upsert high item: %v", err)
	}

	items, err := feed.List(10)
	if err != nil {
		t.Fatalf("list feed: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ArticleID != "high" {
		t.Fatalf("expected highest score first, got %s", items[0].ArticleID)
	}
}

func TestUpsertScoredItemCarriesScoreReasons(t *testing.T) {
	feed := NewMemoryFeed()
	if err := feed.UpsertScoredItem(eventsv1.FeedItemScoredEvent{
		ArticleID:    "reasoned",
		Title:        "Reasoned score",
		SourceName:   "vc",
		URL:          "https://vc.ru/reasoned",
		Score:        50,
		ScoreReasons: []string{"freshness", "source_diversity"},
	}); err != nil {
		t.Fatalf("upsert item: %v", err)
	}

	items, err := feed.List(10)
	if err != nil {
		t.Fatalf("list feed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if len(items[0].ScoreReasons) != 2 || items[0].ScoreReasons[0] != "freshness" {
		t.Fatalf("unexpected score reasons: %#v", items[0].ScoreReasons)
	}
}

func TestListBalancesScoredItemsAcrossSources(t *testing.T) {
	feed := NewMemoryFeed()
	events := []eventsv1.FeedItemScoredEvent{
		{ArticleID: "habr-1", Title: "Habr 1", SourceName: "habr", URL: "https://habr.com/1", Score: 100, PublishedAt: time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)},
		{ArticleID: "habr-2", Title: "Habr 2", SourceName: "habr", URL: "https://habr.com/2", Score: 99, PublishedAt: time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)},
		{ArticleID: "habr-3", Title: "Habr 3", SourceName: "habr", URL: "https://habr.com/3", Score: 98, PublishedAt: time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)},
		{ArticleID: "vc-1", Title: "VC 1", SourceName: "vc", URL: "https://vc.ru/1", Score: 70, PublishedAt: time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)},
		{ArticleID: "vc-2", Title: "VC 2", SourceName: "vc", URL: "https://vc.ru/2", Score: 69, PublishedAt: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)},
	}
	for _, event := range events {
		if err := feed.UpsertScoredItem(event); err != nil {
			t.Fatalf("upsert item: %v", err)
		}
	}

	items, err := feed.List(4)

	if err != nil {
		t.Fatalf("list feed: %v", err)
	}
	sources := []string{items[0].SourceName, items[1].SourceName, items[2].SourceName, items[3].SourceName}
	expected := []string{"habr", "vc", "habr", "vc"}
	for index := range expected {
		if sources[index] != expected[index] {
			t.Fatalf("expected balanced sources %#v, got %#v", expected, sources)
		}
	}
}
