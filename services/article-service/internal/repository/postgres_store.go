package repository

import (
	"context"
	"database/sql"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
)

type SQLExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type PostgresArticleStore struct {
	execer SQLExecer
}

func NewPostgresArticleStore(execer SQLExecer) *PostgresArticleStore {
	return &PostgresArticleStore{execer: execer}
}

func (store *PostgresArticleStore) Save(ctx context.Context, article articlev1.Article) (articlev1.Article, error) {
	query, args := BuildUpsertArticleQuery(article)
	if _, err := store.execer.ExecContext(ctx, query, args...); err != nil {
		return articlev1.Article{}, err
	}
	return article, nil
}
