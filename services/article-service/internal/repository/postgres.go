package repository

import articlev1 "github.com/hanq/articleflow/contracts/article/v1"

func BuildUpsertArticleQuery(article articlev1.Article) (string, []any) {
	query := `
INSERT INTO articles (
    id,
    source_name,
    external_id,
    url,
    title,
    summary,
    content,
    author,
    tags,
    language,
    published_at,
    parsed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (url) DO UPDATE SET
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    content = EXCLUDED.content,
    author = EXCLUDED.author,
    tags = EXCLUDED.tags,
    language = EXCLUDED.language,
    published_at = EXCLUDED.published_at,
    parsed_at = EXCLUDED.parsed_at,
    updated_at = now()
`
	args := []any{
		article.ID,
		article.SourceName,
		article.ExternalID,
		article.URL,
		article.Title,
		article.Summary,
		article.Content,
		article.Author,
		article.Tags,
		article.Language,
		article.PublishedAt,
		article.ParsedAt,
	}
	return query, args
}

