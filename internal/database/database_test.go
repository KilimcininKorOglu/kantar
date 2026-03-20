package database

import (
	"context"
	"testing"

	"github.com/KilimcininKorOglu/kantar/migrations"
)

func TestNewTestDB(t *testing.T) {
	db := NewTestDB(t)

	if err := db.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	if db.Type() != "postgres" {
		t.Errorf("expected postgres, got %s", db.Type())
	}
}

func TestMigrations(t *testing.T) {
	db := NewTestDB(t)

	// Verify tables exist by inserting and querying
	ctx := context.Background()

	// Insert a user
	_, err := db.Conn().ExecContext(ctx,
		"INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)",
		"admin", "hash123", "super_admin",
	)
	if err != nil {
		t.Fatalf("inserting user: %v", err)
	}

	// Query the user back
	var username, role string
	err = db.Conn().QueryRowContext(ctx,
		"SELECT username, role FROM users WHERE username = $1", "admin",
	).Scan(&username, &role)
	if err != nil {
		t.Fatalf("querying user: %v", err)
	}
	if username != "admin" || role != "super_admin" {
		t.Errorf("got user (%s, %s), want (admin, super_admin)", username, role)
	}

	// Verify packages table
	_, err = db.Conn().ExecContext(ctx,
		"INSERT INTO packages (registry_type, name, status) VALUES ($1, $2, $3)",
		"npm", "express", "approved",
	)
	if err != nil {
		t.Fatalf("inserting package: %v", err)
	}

	var pkgName, status string
	err = db.Conn().QueryRowContext(ctx,
		"SELECT name, status FROM packages WHERE registry_type = $1 AND name = $2", "npm", "express",
	).Scan(&pkgName, &status)
	if err != nil {
		t.Fatalf("querying package: %v", err)
	}
	if pkgName != "express" || status != "approved" {
		t.Errorf("got package (%s, %s), want (express, approved)", pkgName, status)
	}

	// Verify foreign key: package_versions references packages
	var pkgID int64
	err = db.Conn().QueryRowContext(ctx,
		"SELECT id FROM packages WHERE name = $1", "express",
	).Scan(&pkgID)
	if err != nil {
		t.Fatalf("getting package id: %v", err)
	}

	_, err = db.Conn().ExecContext(ctx,
		"INSERT INTO package_versions (package_id, version, size) VALUES ($1, $2, $3)",
		pkgID, "4.18.2", 215000,
	)
	if err != nil {
		t.Fatalf("inserting version: %v", err)
	}

	// Verify audit_logs table
	_, err = db.Conn().ExecContext(ctx,
		"INSERT INTO audit_logs (event, actor_username, result) VALUES ($1, $2, $3)",
		"package.approve", "admin", "success",
	)
	if err != nil {
		t.Fatalf("inserting audit log: %v", err)
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	// Running migrations again should not fail
	if err := db.MigrateWithFS(ctx, migrations.FS); err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}

	// And a third time
	if err := db.MigrateWithFS(ctx, migrations.FS); err != nil {
		t.Fatalf("third migration run failed: %v", err)
	}
}

func TestParallelTestDBs(t *testing.T) {
	// Verify that multiple test DBs can be created in parallel
	t.Run("db1", func(t *testing.T) {
		t.Parallel()
		db := NewTestDB(t)
		_, err := db.Conn().ExecContext(context.Background(),
			"INSERT INTO users (username, password_hash) VALUES ($1, $2)", "user1", "hash1")
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	})

	t.Run("db2", func(t *testing.T) {
		t.Parallel()
		db := NewTestDB(t)
		_, err := db.Conn().ExecContext(context.Background(),
			"INSERT INTO users (username, password_hash) VALUES ($1, $2)", "user2", "hash2")
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	})
}
