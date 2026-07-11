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
	profile, ok := store.GetProfile("reader-1")
	if !ok || profile.ID != "reader-1" {
		t.Fatalf("expected reaction to ensure user profile, got %#v ok=%v", profile, ok)
	}
}

func TestMemoryReactionsEnsuresUserProfile(t *testing.T) {
	store := NewMemoryReactions()
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)

	profile, err := store.EnsureProfile(userv1.UserProfile{
		ID:        "reader-1",
		Interests: []string{"go", "kafka"},
		CreatedAt: now,
	})

	if err != nil {
		t.Fatalf("ensure profile: %v", err)
	}
	if profile.ID != "reader-1" || len(profile.Interests) != 2 {
		t.Fatalf("unexpected profile: %#v", profile)
	}
	stored, ok := store.GetProfile("reader-1")
	if !ok || !stored.CreatedAt.Equal(now) {
		t.Fatalf("expected stored profile, got %#v ok=%v", stored, ok)
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
