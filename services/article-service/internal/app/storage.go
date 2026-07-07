package app

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/hanq/articleflow/services/article-service/internal/config"
	"github.com/hanq/articleflow/services/article-service/internal/repository"
	"github.com/hanq/articleflow/services/article-service/internal/usecase"
)

type databaseHandle interface {
	repository.SQLExecer
	PingContext(ctx context.Context) error
	Close() error
}

type databaseOpener func(driverName string, dsn string) (databaseHandle, error)

func openDatabase(driverName string, dsn string) (databaseHandle, error) {
	return sql.Open(driverName, dsn)
}

func newArticleStore(ctx context.Context, cfg config.Config, opener databaseOpener) (usecase.ArticleStore, func() error, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.StorageDriver))
	if driver == "" || driver == "memory" {
		return usecase.NewMemoryArticleStore(), noopClose, nil
	}
	if driver != "postgres" {
		return nil, noopClose, errors.New("unsupported article storage driver")
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
	return repository.NewPostgresArticleStore(database), database.Close, nil
}

func noopClose() error {
	return nil
}
