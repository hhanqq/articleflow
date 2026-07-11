package repository

import (
	"context"
	"database/sql"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type SQLExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type SQLQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type PostgresReactionStore struct {
	database interface {
		SQLExecer
		SQLQueryer
	}
}

func NewPostgresReactionStore(database interface {
	SQLExecer
	SQLQueryer
}) *PostgresReactionStore {
	return &PostgresReactionStore{database: database}
}

func (store *PostgresReactionStore) Record(reaction userv1.UserReaction) error {
	const query = `
INSERT INTO user_reactions (
    user_id,
    article_id,
    type,
    created_at
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (user_id, article_id, type) DO UPDATE SET
    created_at = EXCLUDED.created_at,
    updated_at = now()
`
	_, err := store.database.ExecContext(
		context.Background(),
		query,
		reaction.UserID,
		reaction.ArticleID,
		string(reaction.Type),
		reaction.CreatedAt,
	)
	return err
}

func (store *PostgresReactionStore) ListByUser(userID string) []userv1.UserReaction {
	const query = `
SELECT user_id, article_id, type, created_at
FROM user_reactions
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 500
`
	rows, err := store.database.QueryContext(context.Background(), query, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	reactions := make([]userv1.UserReaction, 0)
	for rows.Next() {
		var reaction userv1.UserReaction
		var reactionType string
		if err := rows.Scan(&reaction.UserID, &reaction.ArticleID, &reactionType, &reaction.CreatedAt); err != nil {
			return reactions
		}
		reaction.Type = userv1.ReactionType(reactionType)
		reactions = append(reactions, reaction)
	}
	return reactions
}
