package usecase

import (
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

func TestSignalStoreBoostsArticlesWithPositiveReactionTags(t *testing.T) {
	signals := NewSignalStore()
	signals.RememberArticle(feedv1.FeedItem{
		ArticleID: "habr:go-kafka",
		Title:     "Go Kafka",
		Tags:      []string{"go", "kafka"},
	})

	err := signals.RecordReaction(eventsv1.UserReactionCreatedEvent{
		UserID:    "reader-1",
		ArticleID: "habr:go-kafka",
		Type:      string(userv1.ReactionSave),
		CreatedAt: time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("record reaction: %v", err)
	}
	base := 10.0
	boosted := signals.ApplyToScore(base, feedv1.FeedItem{
		ArticleID: "habr:go-concurrency",
		Title:     "Go concurrency",
		Tags:      []string{"go"},
	})
	neutral := signals.ApplyToScore(base, feedv1.FeedItem{
		ArticleID: "habr:css",
		Title:     "CSS layout",
		Tags:      []string{"css"},
	})

	if boosted <= neutral {
		t.Fatalf("expected tag boost, got boosted=%f neutral=%f", boosted, neutral)
	}
}

func TestSignalStorePenalizesArticlesWithNegativeReactionTags(t *testing.T) {
	signals := NewSignalStore()
	signals.RememberArticle(feedv1.FeedItem{
		ArticleID: "habr:css",
		Title:     "CSS layout",
		Tags:      []string{"css"},
	})

	err := signals.RecordReaction(eventsv1.UserReactionCreatedEvent{
		UserID:    "reader-1",
		ArticleID: "habr:css",
		Type:      string(userv1.ReactionDislike),
		CreatedAt: time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("record reaction: %v", err)
	}
	score := signals.ApplyToScore(10, feedv1.FeedItem{
		ArticleID: "habr:css-grid",
		Title:     "CSS grid",
		Tags:      []string{"css"},
	})

	if score >= 10 {
		t.Fatalf("expected penalty below base score, got %f", score)
	}
}

func TestSignalStoreBoostsPreferredSources(t *testing.T) {
	signals := NewSignalStore()
	signals.RememberArticle(feedv1.FeedItem{
		ArticleID:  "dzen:travel",
		SourceName: "dzen",
		Title:      "Маршрут по Турции",
	})

	err := signals.RecordReaction(eventsv1.UserReactionCreatedEvent{
		UserID:    "reader-1",
		ArticleID: "dzen:travel",
		Type:      string(userv1.ReactionSave),
		CreatedAt: time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("record reaction: %v", err)
	}
	base := 10.0
	boosted := signals.ApplyToScore(base, feedv1.FeedItem{
		ArticleID:  "dzen:next",
		SourceName: "dzen",
		Title:      "Новая статья без тегов",
	})
	neutral := signals.ApplyToScore(base, feedv1.FeedItem{
		ArticleID:  "habr:next",
		SourceName: "habr",
		Title:      "Новая статья без тегов",
	})

	if boosted <= neutral {
		t.Fatalf("expected source boost, got boosted=%f neutral=%f", boosted, neutral)
	}
}
