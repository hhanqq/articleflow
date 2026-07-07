package usecase

import (
	"testing"
	"time"

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
}

