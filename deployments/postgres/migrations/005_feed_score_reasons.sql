ALTER TABLE feed_items
    ADD COLUMN IF NOT EXISTS score_reasons text[] NOT NULL DEFAULT '{}';
