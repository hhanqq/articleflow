package repository

import (
	"context"
	"database/sql"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type SQLExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type PostgresReactionStore struct {
	database SQLExecer
}

func NewPostgresReactionStore(database SQLExecer) *PostgresReactionStore {
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
