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
	query string
	args  []any
}

func (database *recordingDatabase) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	database.query = query
	database.args = args
	return fakeResult{}, nil
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

	if !strings.Contains(database.query, "INSERT INTO user_reactions") {
		t.Fatalf("expected insert query, got %s", database.query)
	}
	if !strings.Contains(database.query, "ON CONFLICT") {
		t.Fatalf("expected upsert query, got %s", database.query)
	}
	if len(database.args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(database.args))
	}
	if database.args[0] != "reader-1" || database.args[1] != "habr:123" {
		t.Fatalf("unexpected args: %#v", database.args)
	}
}
