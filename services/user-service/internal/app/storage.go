package app

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/hanq/articleflow/services/user-service/internal/config"
	"github.com/hanq/articleflow/services/user-service/internal/repository"
	"github.com/hanq/articleflow/services/user-service/internal/usecase"
)

type databaseHandle interface {
	repository.SQLExecer
	repository.SQLQueryer
	PingContext(ctx context.Context) error
	Close() error
}

type databaseOpener func(driverName string, dsn string) (databaseHandle, error)

func openDatabase(driverName string, dsn string) (databaseHandle, error) {
	return sql.Open(driverName, dsn)
}

func newReactionStore(ctx context.Context, cfg config.Config, opener databaseOpener) (usecase.ReactionStore, func() error, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.StorageDriver))
	if driver == "" {
		driver = "postgres"
	}
	if driver == "memory" {
		return usecase.NewMemoryReactions(), noopClose, nil
	}
	if driver != "postgres" {
		return nil, noopClose, errors.New("unsupported user storage driver")
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
	return repository.NewPostgresReactionStore(database), database.Close, nil
}

func noopClose() error {
	return nil
}
