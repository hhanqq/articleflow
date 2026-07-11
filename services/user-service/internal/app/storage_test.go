package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/hanq/articleflow/services/user-service/internal/config"
	"github.com/hanq/articleflow/services/user-service/internal/repository"
	"github.com/hanq/articleflow/services/user-service/internal/usecase"
)

type fakeDatabase struct {
	pinged bool
	closed bool
	err    error
}

func (database *fakeDatabase) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}

func (database *fakeDatabase) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (database *fakeDatabase) PingContext(context.Context) error {
	database.pinged = true
	return database.err
}

func (database *fakeDatabase) Close() error {
	database.closed = true
	return nil
}

func TestNewReactionStoreReturnsMemoryStoreWhenExplicit(t *testing.T) {
	store, closeStore, err := newReactionStore(context.Background(), config.Config{StorageDriver: "memory"}, nil)
	defer closeStore()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := store.(*usecase.MemoryReactions); !ok {
		t.Fatalf("expected memory store, got %T", store)
	}
}

func TestNewReactionStoreDefaultsToPostgres(t *testing.T) {
	database := &fakeDatabase{}
	store, closeStore, err := newReactionStore(context.Background(), config.Config{
		PostgresDSN: "postgres://articleflow",
	}, func(driverName string, dsn string) (databaseHandle, error) {
		if driverName != "pgx" {
			t.Fatalf("expected pgx driver, got %s", driverName)
		}
		if dsn == "" {
			t.Fatal("expected dsn")
		}
		return database, nil
	})
	defer closeStore()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := store.(*repository.PostgresReactionStore); !ok {
		t.Fatalf("expected postgres store, got %T", store)
	}
	if !database.pinged {
		t.Fatal("expected database ping")
	}
}

func TestNewReactionStoreReturnsPostgresPingError(t *testing.T) {
	_, closeStore, err := newReactionStore(context.Background(), config.Config{
		StorageDriver: "postgres",
		PostgresDSN:   "postgres://articleflow",
	}, func(string, string) (databaseHandle, error) {
		return &fakeDatabase{err: errors.New("postgres down")}, nil
	})
	defer closeStore()

	if err == nil {
		t.Fatal("expected ping error")
	}
}
