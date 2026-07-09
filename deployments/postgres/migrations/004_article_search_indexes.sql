-- +goose Up
CREATE INDEX IF NOT EXISTS idx_articles_search_vector ON articles USING GIN ((
    setweight(to_tsvector('russian', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('russian', coalesce(summary, '')), 'B') ||
    setweight(to_tsvector('russian', coalesce(content, '')), 'C') ||
    setweight(to_tsvector('russian', coalesce(author, '')), 'D')
));

CREATE INDEX IF NOT EXISTS idx_articles_tags_gin ON articles USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_articles_source_published_at ON articles (source_name, published_at DESC);
