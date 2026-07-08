package app

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/jobs"
)

type databaseHandle interface {
	jobs.SQLJobDatabase
	PingContext(ctx context.Context) error
	Close() error
}

type databaseOpener func(driverName string, dsn string) (databaseHandle, error)

func openDatabase(driverName string, dsn string) (databaseHandle, error) {
	return sql.Open(driverName, dsn)
}

func newJobStore(ctx context.Context, cfg config.Config, opener databaseOpener) (jobs.Store, func() error, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.StorageDriver))
	if driver == "" {
		driver = "postgres"
	}
	if driver == "memory" {
		return jobs.NewMemoryStore(), noopClose, nil
	}
	if driver != "postgres" {
		return nil, noopClose, errors.New("unsupported parser storage driver")
	}
	if opener == nil {
		opener = openDatabase
	}
	database, err := opener("pgx", cfg.PostgresDSN)
	if err != nil {
		return nil, noopClose, err
	}
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, noopClose, err
	}
	return jobs.NewPostgresStore(database), database.Close, nil
}

func noopClose() error {
	return nil
}
