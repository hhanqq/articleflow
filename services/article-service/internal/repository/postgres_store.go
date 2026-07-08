package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
)

type SQLExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type SQLQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
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

func (store *PostgresArticleStore) Search(ctx context.Context, query articlev1.SearchQuery) ([]articlev1.Article, error) {
	query = query.Normalize()
	if query.Text == "" {
		return []articlev1.Article{}, nil
	}
	sqlQuery, args := BuildSearchArticlesQuery(query)
	rows, err := store.queryer.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := make([]articlev1.Article, 0, query.Limit)
	for rows.Next() {
		article, err := scanArticleRows(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return articles, nil
}

func BuildSearchArticlesQuery(query articlev1.SearchQuery) (string, []any) {
	query = query.Normalize()
	args := []any{query.Text}
	searchVector := articleSearchVector()
	searchQuery := "websearch_to_tsquery('russian', $1)"
	clauses := []string{fmt.Sprintf("%s @@ %s", searchVector, searchQuery)}

	sources := cleanLowerStrings(query.Sources)
	if len(sources) > 0 {
		placeholders := make([]string, 0, len(sources))
		for _, source := range sources {
			args = append(args, source)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		clauses = append(clauses, "lower(source_name) IN ("+strings.Join(placeholders, ", ")+")")
	}

	for _, tag := range cleanLowerStrings(query.Tags) {
		args = append(args, tag)
		clauses = append(clauses, fmt.Sprintf("EXISTS (SELECT 1 FROM unnest(tags) AS article_tag WHERE lower(btrim(trim(trailing '*' from article_tag))) = $%d)", len(args)))
	}
	if query.FromDate != nil {
		args = append(args, *query.FromDate)
		clauses = append(clauses, fmt.Sprintf("published_at >= $%d", len(args)))
	}
	if query.ToDate != nil {
		args = append(args, *query.ToDate)
		clauses = append(clauses, fmt.Sprintf("published_at <= $%d", len(args)))
	}

	args = append(args, query.Limit)
	limitPlaceholder := len(args)
	args = append(args, query.Offset)
	offsetPlaceholder := len(args)

	return `
SELECT id, source_name, external_id, url, title, summary, content, author, array_to_json(tags)::text, language, published_at, parsed_at
FROM articles
WHERE ` + strings.Join(clauses, " AND ") + `
ORDER BY ts_rank_cd(` + searchVector + `, ` + searchQuery + `) DESC, published_at DESC NULLS LAST, parsed_at DESC
LIMIT $` + fmt.Sprint(limitPlaceholder) + `
OFFSET $` + fmt.Sprint(offsetPlaceholder) + `
`, args
}

func articleSearchVector() string {
	return `(
	setweight(to_tsvector('russian', coalesce(title, '')), 'A') ||
	setweight(to_tsvector('russian', coalesce(summary, '')), 'B') ||
	setweight(to_tsvector('russian', coalesce(content, '')), 'C') ||
	setweight(to_tsvector('russian', coalesce(author, '')), 'D')
)`
}

func cleanLowerStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}
	return cleaned
}

type articleScanner interface {
	Scan(dest ...any) error
}

func scanArticleRows(scanner articleScanner) (articlev1.Article, error) {
	var article articlev1.Article
	var tagsPayload string
	var publishedAt sql.NullTime
	var parsedAt time.Time
	err := scanner.Scan(
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
	if err != nil {
		return articlev1.Article{}, err
	}
	if publishedAt.Valid {
		article.PublishedAt = publishedAt.Time
	}
	if tagsPayload != "" {
		if err := json.Unmarshal([]byte(tagsPayload), &article.Tags); err != nil {
			return articlev1.Article{}, err
		}
	}
	article.ParsedAt = parsedAt
	return article, nil
}
