package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

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
	if _, err := store.EnsureProfile(userv1.UserProfile{
		ID:        reaction.UserID,
		CreatedAt: reaction.CreatedAt,
	}); err != nil {
		return err
	}
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

func (store *PostgresReactionStore) EnsureProfile(profile userv1.UserProfile) (userv1.UserProfile, error) {
	profile.ID = strings.TrimSpace(profile.ID)
	if profile.ID == "" {
		return userv1.UserProfile{}, sql.ErrNoRows
	}
	profile.Email = strings.TrimSpace(profile.Email)
	profile.Interests = append([]string(nil), profile.Interests...)
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now().UTC()
	}
	interestsPayload, err := json.Marshal(profile.Interests)
	if err != nil {
		return userv1.UserProfile{}, err
	}
	const query = `
INSERT INTO users (
    id,
    email,
    interests,
    created_at
) VALUES (
    $1, $2, $3::jsonb, $4
)
ON CONFLICT (id) DO UPDATE SET
    email = CASE WHEN EXCLUDED.email <> '' THEN EXCLUDED.email ELSE users.email END,
    interests = CASE WHEN EXCLUDED.interests <> '[]'::jsonb THEN EXCLUDED.interests ELSE users.interests END,
    updated_at = now()
`
	_, err = store.database.ExecContext(
		context.Background(),
		query,
		profile.ID,
		profile.Email,
		string(interestsPayload),
		profile.CreatedAt,
	)
	if err != nil {
		return userv1.UserProfile{}, err
	}
	return profile, nil
}

func (store *PostgresReactionStore) GetProfile(userID string) (userv1.UserProfile, bool) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return userv1.UserProfile{}, false
	}
	const query = `
SELECT id, email, interests::text, created_at
FROM users
WHERE id = $1
LIMIT 1
`
	rows, err := store.database.QueryContext(context.Background(), query, userID)
	if err != nil {
		return userv1.UserProfile{}, false
	}
	defer rows.Close()
	if !rows.Next() {
		return userv1.UserProfile{}, false
	}
	var profile userv1.UserProfile
	var interestsPayload string
	if err := rows.Scan(&profile.ID, &profile.Email, &interestsPayload, &profile.CreatedAt); err != nil {
		return userv1.UserProfile{}, false
	}
	if err := json.Unmarshal([]byte(interestsPayload), &profile.Interests); err != nil {
		profile.Interests = nil
	}
	return profile, true
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
