package repository

import (
	"encoding/json"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
)

func BuildUpsertFeedItemQuery(event eventsv1.FeedItemScoredEvent) (string, []any) {
	tags := event.Tags
	if tags == nil {
		tags = []string{}
	}
	query := `
INSERT INTO feed_items (
    article_id,
    source_name,
    url,
    title,
    summary,
    tags,
    score,
    published_at,
    scored_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (article_id) DO UPDATE SET
    source_name = EXCLUDED.source_name,
    url = EXCLUDED.url,
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    tags = EXCLUDED.tags,
    score = EXCLUDED.score,
    published_at = EXCLUDED.published_at,
    scored_at = EXCLUDED.scored_at,
    updated_at = now()
`
	args := []any{
		event.ArticleID,
		event.SourceName,
		event.URL,
		event.Title,
		event.Summary,
		tags,
		event.Score,
		event.PublishedAt,
		event.ScoredAt,
	}
	return query, args
}

func BuildListFeedItemsQuery(limit int) (string, []any) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	query := `
WITH ranked_feed_items AS (
  SELECT
    article_id,
    title,
    summary,
    source_name,
    url,
    tags,
    score,
    published_at,
    ROW_NUMBER() OVER (PARTITION BY lower(source_name) ORDER BY score DESC, published_at DESC NULLS LAST) AS source_rank
  FROM feed_items
)
SELECT article_id, title, summary, source_name, url, array_to_json(tags)::text, score, published_at
FROM ranked_feed_items
ORDER BY source_rank ASC, score DESC, published_at DESC NULLS LAST
LIMIT $1
`
	return query, []any{limit}
}

func DecodeTagsJSON(payload string) ([]string, error) {
	if payload == "" {
		return nil, nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(payload), &tags); err != nil {
		return nil, err
	}
	return tags, nil
}
