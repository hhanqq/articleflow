-- +goose Up
CREATE TABLE IF NOT EXISTS user_reactions (
    user_id TEXT NOT NULL,
    article_id TEXT NOT NULL,
    type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, article_id, type)
);

CREATE INDEX IF NOT EXISTS idx_user_reactions_article_id ON user_reactions (article_id);
CREATE INDEX IF NOT EXISTS idx_user_reactions_created_at ON user_reactions (created_at DESC);
