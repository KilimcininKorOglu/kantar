package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

// MigrateWithFS runs migrations using the provided filesystem.
// Migration files must be named NNN_description.sql (e.g., 001_initial_schema.sql).
func (db *DB) MigrateWithFS(ctx context.Context, migrations fs.FS) error {
	return runMigrations(ctx, db.conn, migrations)
}

func runMigrations(ctx context.Context, conn *sql.DB, migrations fs.FS) error {
	// Ensure schema_migrations table exists
	_, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	// Get already applied versions
	applied, err := getAppliedVersions(ctx, conn)
	if err != nil {
		return fmt.Errorf("checking applied migrations: %w", err)
	}

	// Find migration files
	entries, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		return fmt.Errorf("reading migration files: %w", err)
	}
	sort.Strings(entries)

	for _, entry := range entries {
		version := extractVersion(entry)
		if version == 0 {
			continue
		}

		if applied[version] {
			continue
		}

		content, err := fs.ReadFile(migrations, entry)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", entry, err)
		}

		sqlStr := string(content)

		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("starting transaction for migration %d: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, sqlStr); err != nil {
			tx.Rollback()
			return fmt.Errorf("applying migration %d: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording migration %d: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration %d: %w", version, err)
		}
	}

	return nil
}

func getAppliedVersions(ctx context.Context, conn *sql.DB) (map[int]bool, error) {
	rows, err := conn.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func extractVersion(filename string) int {
	// "001_initial_schema.sql" → 1
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) == 0 {
		return 0
	}
	v, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return v
}
