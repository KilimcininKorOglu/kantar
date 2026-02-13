// Package database provides the database abstraction layer for Kantar.
package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/KilimcininKorOglu/kantar/internal/config"
)

// DB wraps the database connection and provides access to generated queries.
type DB struct {
	conn *sql.DB
	cfg  config.DatabaseConfig
}

// New creates a new database connection based on the configuration.
func New(cfg config.DatabaseConfig) (*DB, error) {
	var conn *sql.DB
	var err error

	switch cfg.Type {
	case "sqlite":
		conn, err = openSQLite(cfg.Path)
	case "postgres":
		conn, err = openPostgres(cfg.Postgres)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("opening database (%s): %w", cfg.Type, err)
	}

	// Verify connectivity
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{conn: conn, cfg: cfg}, nil
}

// Conn returns the underlying *sql.DB connection.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Ping verifies the database connection is alive.
func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

// Type returns the database type (sqlite or postgres).
func (db *DB) Type() string {
	return db.cfg.Type
}
