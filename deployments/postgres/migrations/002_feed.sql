-- +goose Up
CREATE TABLE IF NOT EXISTS feed_items (
    article_id TEXT PRIMARY KEY,
    source_name TEXT NOT NULL,
    url TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    score DOUBLE PRECISION NOT NULL DEFAULT 0,
    published_at TIMESTAMPTZ,
    scored_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_feed_items_score ON feed_items (score DESC);
CREATE INDEX IF NOT EXISTS idx_feed_items_published_at ON feed_items (published_at DESC);
