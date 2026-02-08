package utils

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func SetupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:18-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	var db *sqlx.DB
	const maxRetries = 10
	for i := 0; i < maxRetries; i++ {
		db, err = sqlx.Connect("postgres", connStr)
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if err != nil {
		t.Fatalf("failed to connect to test database after %d retries: %v", maxRetries, err)
	}

	_, thisFile, _, _ := runtime.Caller(0)
	initSQLPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "sql", "init.sql")
	initSQL, err := os.ReadFile(initSQLPath)
	if err != nil {
		db.Close()
		t.Fatalf("failed to read schema file: %v", err)
	}

	if _, err := db.Exec(string(initSQL)); err != nil {
		db.Close()
		t.Fatalf("failed to run schema migrations: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate postgres container: %v", err)
		}
	})

	return db
}
