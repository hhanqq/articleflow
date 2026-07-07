package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
)

type SQLExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type SQLQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type PostgresArticleStore struct {
	execer  SQLExecer
	queryer SQLQueryer
}

type SQLArticleDatabase interface {
	SQLExecer
	SQLQueryer
}

func NewPostgresArticleStore(database SQLArticleDatabase) *PostgresArticleStore {
	return &PostgresArticleStore{execer: database, queryer: database}
}

func (store *PostgresArticleStore) Save(ctx context.Context, article articlev1.Article) (articlev1.Article, error) {
	query, args := BuildUpsertArticleQuery(article)
	if _, err := store.execer.ExecContext(ctx, query, args...); err != nil {
		return articlev1.Article{}, err
	}
	return article, nil
}

func (store *PostgresArticleStore) GetByID(ctx context.Context, id string) (articlev1.Article, bool, error) {
	const query = `
SELECT id, source_name, external_id, url, title, summary, content, author, array_to_json(tags)::text, language, published_at, parsed_at
FROM articles
WHERE id = $1
`
	var article articlev1.Article
	var tagsPayload string
	var publishedAt sql.NullTime
	var parsedAt time.Time
	err := store.queryer.QueryRowContext(ctx, query, id).Scan(
		&article.ID,
		&article.SourceName,
		&article.ExternalID,
		&article.URL,
		&article.Title,
		&article.Summary,
		&article.Content,
		&article.Author,
		&tagsPayload,
		&article.Language,
		&publishedAt,
		&parsedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return articlev1.Article{}, false, nil
	}
	if err != nil {
		return articlev1.Article{}, false, err
	}
	if publishedAt.Valid {
		article.PublishedAt = publishedAt.Time
	}
	if tagsPayload != "" {
		if err := json.Unmarshal([]byte(tagsPayload), &article.Tags); err != nil {
			return articlev1.Article{}, false, err
		}
	}
	article.ParsedAt = parsedAt
	return article, true, nil
}
