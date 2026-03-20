package database

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/KilimcininKorOglu/kantar/internal/config"
	"github.com/KilimcininKorOglu/kantar/migrations"
)

// NewTestDB creates a PostgreSQL test database.
// It reads the connection from TEST_DATABASE_URL env var.
// If not set, defaults to a local PostgreSQL instance.
// The database is automatically migrated and cleaned up when the test ends.
func NewTestDB(t *testing.T) *DB {
	t.Helper()

	host := envOrDefault("TEST_DB_HOST", "localhost")
	port := envOrDefault("TEST_DB_PORT", "5432")
	user := envOrDefault("TEST_DB_USER", "kantar")
	password := envOrDefault("TEST_DB_PASSWORD", "kantar")
	name := envOrDefault("TEST_DB_NAME", fmt.Sprintf("kantar_test_%d", os.Getpid()))

	portInt := 5432
	fmt.Sscanf(port, "%d", &portInt)

	cfg := config.DatabaseConfig{
		Type: "postgres",
		Postgres: config.PostgresConfig{
			Host:     host,
			Port:     portInt,
			Name:     name,
			User:     user,
			Password: password,
			SSLMode:  "disable",
		},
	}

	// Try to connect to the default "postgres" database to create the test DB
	setupCfg := cfg
	setupCfg.Postgres.Name = "postgres"
	setupDB, err := New(setupCfg)
	if err != nil {
		t.Skipf("skipping test: PostgreSQL not available: %v", err)
	}

	// Create test database (ignore error if already exists)
	_, _ = setupDB.conn.Exec(fmt.Sprintf("CREATE DATABASE %s", name))
	setupDB.Close()

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
		// Drop test database
		cleanupDB, cleanupErr := New(setupCfg)
		if cleanupErr == nil {
			cleanupDB.conn.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))
			cleanupDB.Close()
		}
	})

	return db
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
