package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type SQLJobDatabase interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type PostgresStore struct {
	database SQLJobDatabase
}

func NewPostgresStore(database SQLJobDatabase) *PostgresStore {
	return &PostgresStore{database: database}
}

func (store *PostgresStore) Save(ctx context.Context, job parserv1.ParserJob) (parserv1.ParserJob, error) {
	queryPayload, err := json.Marshal(job.Query)
	if err != nil {
		return parserv1.ParserJob{}, err
	}
	candidatesPayload, err := json.Marshal(job.Candidates)
	if err != nil {
		return parserv1.ParserJob{}, err
	}
	sourceStatsPayload, err := json.Marshal(job.SourceStats)
	if err != nil {
		return parserv1.ParserJob{}, err
	}
	_, err = store.database.ExecContext(ctx, `
INSERT INTO parser_jobs (
    id,
    query,
    status,
    error,
    candidates_count,
    candidates,
    source_stats,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (id) DO UPDATE SET
    query = EXCLUDED.query,
    status = EXCLUDED.status,
    error = EXCLUDED.error,
    candidates_count = EXCLUDED.candidates_count,
    candidates = EXCLUDED.candidates,
    source_stats = EXCLUDED.source_stats,
    updated_at = EXCLUDED.updated_at
`, job.ID, string(queryPayload), job.Status, job.Error, job.CandidatesCount, string(candidatesPayload), string(sourceStatsPayload), job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return parserv1.ParserJob{}, err
	}
	return job, nil
}

func (store *PostgresStore) FindByID(ctx context.Context, id string) (parserv1.ParserJob, bool, error) {
	row := store.database.QueryRowContext(ctx, `
SELECT id, query::text, status, error, candidates_count, candidates::text, source_stats::text, created_at, updated_at
FROM parser_jobs
WHERE id = $1
`, id)
	job, err := scanParserJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return parserv1.ParserJob{}, false, nil
	}
	if err != nil {
		return parserv1.ParserJob{}, false, err
	}
	return job, true, nil
}

func (store *PostgresStore) List(ctx context.Context, limit int) ([]parserv1.ParserJob, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := store.database.QueryContext(ctx, `
SELECT id, query::text, status, error, candidates_count, candidates::text, source_stats::text, created_at, updated_at
FROM parser_jobs
ORDER BY updated_at DESC, created_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]parserv1.ParserJob, 0, limit)
	for rows.Next() {
		job, err := scanParserJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}

type parserJobScanner interface {
	Scan(dest ...any) error
}

func scanParserJob(scanner parserJobScanner) (parserv1.ParserJob, error) {
	var job parserv1.ParserJob
	var queryPayload string
	var candidatesPayload string
	var sourceStatsPayload string
	var createdAt time.Time
	var updatedAt time.Time
	err := scanner.Scan(
		&job.ID,
		&queryPayload,
		&job.Status,
		&job.Error,
		&job.CandidatesCount,
		&candidatesPayload,
		&sourceStatsPayload,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return parserv1.ParserJob{}, err
	}
	if err := json.Unmarshal([]byte(queryPayload), &job.Query); err != nil {
		return parserv1.ParserJob{}, err
	}
	if candidatesPayload != "" {
		if err := json.Unmarshal([]byte(candidatesPayload), &job.Candidates); err != nil {
			return parserv1.ParserJob{}, err
		}
	}
	if sourceStatsPayload != "" {
		if err := json.Unmarshal([]byte(sourceStatsPayload), &job.SourceStats); err != nil {
			return parserv1.ParserJob{}, err
		}
	}
	job.CreatedAt = createdAt
	job.UpdatedAt = updatedAt
	return job, nil
}
