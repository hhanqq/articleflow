CREATE TABLE IF NOT EXISTS parser_jobs (
    id text PRIMARY KEY,
    query jsonb NOT NULL,
    status text NOT NULL,
    error text NOT NULL DEFAULT '',
    candidates_count integer NOT NULL DEFAULT 0,
    candidates jsonb NOT NULL DEFAULT '[]'::jsonb,
    source_stats jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_parser_jobs_updated_at ON parser_jobs (updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_parser_jobs_status ON parser_jobs (status);
