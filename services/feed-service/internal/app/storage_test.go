package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/hanq/articleflow/services/feed-service/internal/config"
)

func TestNewFeedStoreDefaultsToMemory(t *testing.T) {
	store, closeStore, err := newFeedStore(context.Background(), config.Config{StorageDriver: "memory"}, nil)
	if err != nil {
		t.Fatalf("new feed store: %v", err)
	}
	defer closeStore()

	if store == nil {
		t.Fatal("expected store")
	}
}

func TestNewFeedStoreDefaultsToPostgres(t *testing.T) {
	opener := func(driverName string, dsn string) (databaseHandle, error) {
		if driverName != "pgx" {
			t.Fatalf("expected pgx driver, got %s", driverName)
		}
		if dsn == "" {
			t.Fatal("expected dsn")
		}
		return fakeDatabase{}, nil
	}

	store, closeStore, err := newFeedStore(context.Background(), config.Config{
		PostgresDSN: "postgres://articleflow",
	}, opener)
	if err != nil {
		t.Fatalf("new feed store: %v", err)
	}
	defer closeStore()

	if store == nil {
		t.Fatal("expected store")
	}
}

func TestNewFeedStoreRejectsUnsupportedDriver(t *testing.T) {
	_, _, err := newFeedStore(context.Background(), config.Config{StorageDriver: "unknown"}, nil)

	if err == nil {
		t.Fatal("expected unsupported driver error")
	}
}

func TestNewFeedStoreReturnsPostgresPingError(t *testing.T) {
	opener := func(string, string) (databaseHandle, error) {
		return fakeDatabase{pingErr: errors.New("no database")}, nil
	}

	_, _, err := newFeedStore(context.Background(), config.Config{
		StorageDriver: "postgres",
		PostgresDSN:   "postgres://articleflow",
	}, opener)

	if err == nil {
		t.Fatal("expected ping error")
	}
}

type fakeDatabase struct {
	pingErr error
}

func (database fakeDatabase) PingContext(context.Context) error {
	return database.pingErr
}

func (database fakeDatabase) Close() error {
	return nil
}

func (database fakeDatabase) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}

func (database fakeDatabase) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}
