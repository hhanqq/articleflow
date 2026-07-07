package usecase

import (
	"testing"
	"time"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

func TestMemoryReactionsStoresUserReaction(t *testing.T) {
	store := NewMemoryReactions()
	reaction := userv1.UserReaction{
		UserID:    "reader-1",
		ArticleID: "habr:123",
		Type:      userv1.ReactionSave,
		CreatedAt: time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
	}

	if err := store.Record(reaction); err != nil {
		t.Fatalf("record reaction: %v", err)
	}

	reactions := store.ListByUser("reader-1")
	if len(reactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(reactions))
	}
	if reactions[0] != reaction {
		t.Fatalf("unexpected reaction: %+v", reactions[0])
	}
}

func TestMemoryReactionsReplacesSameUserArticleType(t *testing.T) {
	store := NewMemoryReactions()
	first := userv1.UserReaction{
		UserID:    "reader-1",
		ArticleID: "habr:123",
		Type:      userv1.ReactionOpen,
		CreatedAt: time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
	}
	second := userv1.UserReaction{
		UserID:    "reader-1",
		ArticleID: "habr:123",
		Type:      userv1.ReactionOpen,
		CreatedAt: time.Date(2026, 7, 8, 10, 5, 0, 0, time.UTC),
	}

	if err := store.Record(first); err != nil {
		t.Fatalf("record first reaction: %v", err)
	}
	if err := store.Record(second); err != nil {
		t.Fatalf("record second reaction: %v", err)
	}

	reactions := store.ListByUser("reader-1")
	if len(reactions) != 1 {
		t.Fatalf("expected replacement to keep 1 reaction, got %d", len(reactions))
	}
	if !reactions[0].CreatedAt.Equal(second.CreatedAt) {
		t.Fatalf("expected latest reaction time, got %s", reactions[0].CreatedAt)
	}
}
