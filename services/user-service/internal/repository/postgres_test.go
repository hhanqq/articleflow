package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

type recordingDatabase struct {
	queries []string
	args    [][]any
}

func (database *recordingDatabase) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	database.queries = append(database.queries, query)
	database.args = append(database.args, args)
	return fakeResult{}, nil
}

func (database *recordingDatabase) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	return nil, nil
}

func TestPostgresReactionStoreRecordsReaction(t *testing.T) {
	database := &recordingDatabase{}
	store := NewPostgresReactionStore(database)
	reaction := userv1.UserReaction{
		UserID:    "reader-1",
		ArticleID: "habr:123",
		Type:      userv1.ReactionSave,
		CreatedAt: time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
	}

	if err := store.Record(reaction); err != nil {
		t.Fatalf("record reaction: %v", err)
	}

	if len(database.queries) != 2 {
		t.Fatalf("expected profile and reaction upserts, got %d queries", len(database.queries))
	}
	if !strings.Contains(database.queries[0], "INSERT INTO users") {
		t.Fatalf("expected user profile upsert first, got %s", database.queries[0])
	}
	if !strings.Contains(database.queries[1], "INSERT INTO user_reactions") {
		t.Fatalf("expected insert query, got %s", database.queries[1])
	}
	if !strings.Contains(database.queries[1], "ON CONFLICT") {
		t.Fatalf("expected upsert query, got %s", database.queries[1])
	}
	if len(database.args[1]) != 4 {
		t.Fatalf("expected 4 reaction args, got %d", len(database.args[1]))
	}
	if database.args[1][0] != "reader-1" || database.args[1][1] != "habr:123" {
		t.Fatalf("unexpected reaction args: %#v", database.args[1])
	}
}

func TestPostgresReactionStoreEnsuresProfile(t *testing.T) {
	database := &recordingDatabase{}
	store := NewPostgresReactionStore(database)
	profile := userv1.UserProfile{
		ID:        "reader-1",
		Interests: []string{"go"},
		CreatedAt: time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC),
	}

	created, err := store.EnsureProfile(profile)

	if err != nil {
		t.Fatalf("ensure profile: %v", err)
	}
	if created.ID != "reader-1" {
		t.Fatalf("unexpected profile: %#v", created)
	}
	if len(database.queries) != 1 || !strings.Contains(database.queries[0], "INSERT INTO users") {
		t.Fatalf("expected users upsert, got %#v", database.queries)
	}
}
