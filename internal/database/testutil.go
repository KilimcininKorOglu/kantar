package database

import (
	"context"
	"testing"

	"github.com/KilimcininKorOglu/kantar/internal/config"
	"github.com/KilimcininKorOglu/kantar/migrations"
)

// NewTestDB creates an in-memory SQLite database for testing.
// The database is automatically migrated and cleaned up when the test ends.
func NewTestDB(t *testing.T) *DB {
	t.Helper()

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		Path: ":memory:",
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("creating test database: %v", err)
	}

	if err := db.MigrateWithFS(context.Background(), migrations.FS); err != nil {
		db.Close()
		t.Fatalf("migrating test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
