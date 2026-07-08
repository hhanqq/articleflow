package usecase

import (
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

func TestRankBoostsQueryMatchesAndFreshness(t *testing.T) {
	ranker := NewRanker()
	newRelevant := feedv1.FeedItem{
		ArticleID:   "new-relevant",
		Title:       "Go Kafka микросервисы",
		Tags:        []string{"go", "kafka"},
		SourceName:  "habr",
		PublishedAt: time.Date(2026, 7, 7, 10, 30, 0, 0, time.UTC),
	}
	oldIrrelevant := feedv1.FeedItem{
		ArticleID:   "old",
		Title:       "CSS tricks",
		Tags:        []string{"css"},
		SourceName:  "unknown",
		PublishedAt: time.Date(2025, 1, 1, 10, 30, 0, 0, time.UTC),
	}

	ranked := ranker.Rank(parserv1.SearchQuery{Text: "go kafka"}, []feedv1.FeedItem{oldIrrelevant, newRelevant})

	if len(ranked) != 2 {
		t.Fatalf("expected 2 ranked items, got %d", len(ranked))
	}
	if ranked[0].ArticleID != "new-relevant" {
		t.Fatalf("expected relevant item first, got %s", ranked[0].ArticleID)
	}
	if ranked[0].Score <= ranked[1].Score {
		t.Fatalf("expected first score to be higher")
	}
	if len(ranked[0].ScoreReasons) == 0 {
		t.Fatal("expected score reasons for ranked item")
	}
}

func TestRankDiscoveredAddsScoreReasons(t *testing.T) {
	ranker := NewRanker()
	event := eventsv1.ArticleDiscoveredEvent{
		SourceName:   "habr",
		ExternalID:   "123",
		URL:          "https://habr.com/ru/articles/123/",
		Title:        "Go Kafka",
		Summary:      "Streaming microservices",
		Tags:         []string{"go", "kafka"},
		PublishedAt:  time.Now().UTC().Add(-2 * time.Hour),
		DiscoveredAt: time.Now().UTC(),
	}

	scored := ranker.RankDiscovered(event)

	if scored.Score <= 0 {
		t.Fatalf("expected positive score, got %f", scored.Score)
	}
	if len(scored.ScoreReasons) == 0 {
		t.Fatal("expected score reasons")
	}
}
