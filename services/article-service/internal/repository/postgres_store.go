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
	terms := postgresSearchTerms(query.Text)
	if len(terms) == 0 {
		return []articlev1.Article{}, nil
	}
	sqlQuery, args := BuildSearchArticlesQuery(terms, query.Sources, query.Limit)
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

func BuildSearchArticlesQuery(terms []string, sources []string, limit int) (string, []any) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	args := make([]any, 0, len(terms)+len(sources)+1)
	clauses := make([]string, 0, len(terms)+len(sources))
	searchDocument := `lower(
		coalesce(title, '') || ' ' ||
		coalesce(summary, '') || ' ' ||
		coalesce(content, '') || ' ' ||
		coalesce(url, '') || ' ' ||
		coalesce(author, '') || ' ' ||
		coalesce(array_to_string(tags, ' '), '')
	)`
	for _, term := range terms {
		args = append(args, "%"+strings.ToLower(term)+"%")
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", searchDocument, len(args)))
	}
	sourceClauses := make([]string, 0, len(sources))
	for _, source := range sources {
		source = strings.ToLower(strings.TrimSpace(source))
		if source == "" {
			continue
		}
		args = append(args, source)
		sourceClauses = append(sourceClauses, fmt.Sprintf("lower(source_name) = $%d", len(args)))
	}
	if len(sourceClauses) > 0 {
		clauses = append(clauses, "("+strings.Join(sourceClauses, " OR ")+")")
	}
	args = append(args, limit)
	return `
SELECT id, source_name, external_id, url, title, summary, content, author, array_to_json(tags)::text, language, published_at, parsed_at
FROM articles
WHERE ` + strings.Join(clauses, " AND ") + `
ORDER BY published_at DESC NULLS LAST, parsed_at DESC
LIMIT $` + fmt.Sprint(len(args)) + `
`, args
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

func postgresSearchTerms(text string) []string {
	fields := strings.Fields(strings.ToLower(text))
	terms := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.Trim(field, `"'.,:;!?()[]{}<>`)
		if len([]rune(field)) < 2 || postgresSearchStopWord(field) {
			continue
		}
		terms = append(terms, field)
	}
	return terms
}

func postgresSearchStopWord(word string) bool {
	switch word {
	case "и", "в", "во", "на", "по", "с", "со", "о", "об", "от", "до", "для", "из", "за", "к", "ко", "a", "an", "the", "of", "to", "in", "on", "for", "and":
		return true
	default:
		return false
	}
}
