package repository

import (
	"context"
	"database/sql"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
)

type SQLStore interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type PostgresFeedStore struct {
	database SQLStore
}

func NewPostgresFeedStore(database SQLStore) *PostgresFeedStore {
	return &PostgresFeedStore{database: database}
}

func (store *PostgresFeedStore) UpsertScoredItem(event eventsv1.FeedItemScoredEvent) error {
	query, args := BuildUpsertFeedItemQuery(event)
	_, err := store.database.ExecContext(context.Background(), query, args...)
	return err
}

func (store *PostgresFeedStore) List(limit int) ([]feedv1.FeedItem, error) {
	query, args := BuildListFeedItemsQuery(limit)
	rows, err := store.database.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []feedv1.FeedItem
	for rows.Next() {
		var item feedv1.FeedItem
		var tagsPayload string
		var scoreReasonsPayload string
		if err := rows.Scan(
			&item.ArticleID,
			&item.Title,
			&item.Summary,
			&item.SourceName,
			&item.URL,
			&tagsPayload,
			&item.Score,
			&scoreReasonsPayload,
			&item.PublishedAt,
		); err != nil {
			return nil, err
		}
		tags, err := DecodeTagsJSON(tagsPayload)
		if err != nil {
			return nil, err
		}
		item.Tags = tags
		scoreReasons, err := DecodeStringArrayJSON(scoreReasonsPayload)
		if err != nil {
			return nil, err
		}
		item.ScoreReasons = scoreReasons
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
