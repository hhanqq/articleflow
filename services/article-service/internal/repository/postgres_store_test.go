package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
)

type fakeExecResult struct{}

func (fakeExecResult) LastInsertId() (int64, error) { return 0, nil }
func (fakeExecResult) RowsAffected() (int64, error) { return 1, nil }

type recordingExecer struct {
	query string
	args  []any
	err   error
}

func (execer *recordingExecer) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	execer.query = query
	execer.args = args
	if execer.err != nil {
		return nil, execer.err
	}
	return fakeExecResult{}, nil
}

func (execer *recordingExecer) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	return &sql.Row{}
}

func TestPostgresArticleStoreSaveExecutesUpsert(t *testing.T) {
	execer := &recordingExecer{}
	store := NewPostgresArticleStore(execer)
	article := articlev1.Article{
		ID:          "article-1",
		SourceName:  "habr",
		ExternalID:  "habr-1",
		URL:         "https://habr.com/ru/articles/1/",
		Title:       "Go Kafka",
		Tags:        []string{"go", "kafka"},
		Language:    "ru",
		PublishedAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		ParsedAt:    time.Date(2026, 7, 7, 10, 5, 0, 0, time.UTC),
	}

	saved, err := store.Save(context.Background(), article)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if saved.ID != article.ID {
		t.Fatalf("expected saved article id %s, got %s", article.ID, saved.ID)
	}
	if !strings.Contains(execer.query, "ON CONFLICT (url) DO UPDATE") {
		t.Fatalf("expected upsert query, got %s", execer.query)
	}
	if len(execer.args) != 12 {
		t.Fatalf("expected 12 query args, got %d", len(execer.args))
	}
}

func TestPostgresArticleStoreSaveReturnsExecError(t *testing.T) {
	execer := &recordingExecer{err: errors.New("postgres down")}
	store := NewPostgresArticleStore(execer)

	_, err := store.Save(context.Background(), articlev1.Article{ID: "article-1"})

	if err == nil {
		t.Fatal("expected exec error")
	}
}
