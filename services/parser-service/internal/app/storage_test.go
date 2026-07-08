package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/jobs"
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

func (database *fakeDatabase) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return &sql.Row{}
}

func (database *fakeDatabase) PingContext(context.Context) error {
	database.pinged = true
	return database.err
}

func (database *fakeDatabase) Close() error {
	database.closed = true
	return nil
}

func TestNewJobStoreReturnsMemoryStore(t *testing.T) {
	store, closeStore, err := newJobStore(context.Background(), config.Config{StorageDriver: "memory"}, nil)
	defer closeStore()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := store.(*jobs.MemoryStore); !ok {
		t.Fatalf("expected memory store, got %T", store)
	}
}

func TestNewJobStoreDefaultsToPostgresStore(t *testing.T) {
	database := &fakeDatabase{}
	store, closeStore, err := newJobStore(context.Background(), config.Config{
		PostgresDSN: "postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable",
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
	if _, ok := store.(*jobs.PostgresStore); !ok {
		t.Fatalf("expected postgres store, got %T", store)
	}
	if !database.pinged {
		t.Fatal("expected database ping")
	}
}

func TestNewJobStoreReturnsPostgresPingError(t *testing.T) {
	database := &fakeDatabase{err: errors.New("postgres down")}
	_, closeStore, err := newJobStore(context.Background(), config.Config{
		StorageDriver: "postgres",
		PostgresDSN:   "postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable",
	}, func(string, string) (databaseHandle, error) {
		return database, nil
	})
	defer closeStore()

	if err == nil {
		t.Fatal("expected ping error")
	}
	if !database.closed {
		t.Fatal("expected database close on ping error")
	}
}
