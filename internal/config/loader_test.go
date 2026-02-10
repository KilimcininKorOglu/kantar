package config

import (
	"os"
	"testing"
	"time"
)

func TestInterpolateEnvVars(t *testing.T) {
	t.Setenv("TEST_HOST", "myhost")
	t.Setenv("TEST_PORT", "5432")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple variable",
			input:    "host = ${TEST_HOST}",
			expected: "host = myhost",
		},
		{
			name:     "multiple variables",
			input:    "${TEST_HOST}:${TEST_PORT}",
			expected: "myhost:5432",
		},
		{
			name:     "default value used",
			input:    "${MISSING_VAR:-fallback}",
			expected: "fallback",
		},
		{
			name:     "default not used when var exists",
			input:    "${TEST_HOST:-other}",
			expected: "myhost",
		},
		{
			name:     "unset var without default stays as is",
			input:    "${UNSET_VAR}",
			expected: "${UNSET_VAR}",
		},
		{
			name:     "no variables",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interpolateEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDurationUnmarshalText(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"24h", 24 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"365d", 365 * 24 * time.Hour, false},
		{"1h30m", 90 * time.Minute, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var d Duration
			err := d.UnmarshalText([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d.Duration != tt.expected {
				t.Errorf("got %v, want %v", d.Duration, tt.expected)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Setenv("TEST_DB_PASS", "secret123")

	content := `
[server]
host = "127.0.0.1"
port = 9090

[database]
type = "postgres"

[database.postgres]
host = "db.local"
port = 5432
name = "kantar"
user = "kantar"
password = "${TEST_DB_PASS}"
ssl_mode = "require"

[auth]
type = "local"
session_ttl = "12h"
token_ttl = "30d"

[cache]
enabled = true
type = "memory"
max_size = "1GB"
ttl = "30m"
`

	tmpFile, err := os.CreateTemp("", "kantar-*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "127.0.0.1")
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 9090)
	}
	if cfg.Database.Type != "postgres" {
		t.Errorf("Database.Type = %q, want %q", cfg.Database.Type, "postgres")
	}
	if cfg.Database.Postgres.Password != "secret123" {
		t.Errorf("Database.Postgres.Password = %q, want %q", cfg.Database.Postgres.Password, "secret123")
	}
	if cfg.Auth.SessionTTL.Duration != 12*time.Hour {
		t.Errorf("Auth.SessionTTL = %v, want %v", cfg.Auth.SessionTTL.Duration, 12*time.Hour)
	}
	if cfg.Auth.TokenTTL.Duration != 30*24*time.Hour {
		t.Errorf("Auth.TokenTTL = %v, want %v", cfg.Auth.TokenTTL.Duration, 30*24*time.Hour)
	}
	if cfg.Cache.TTL.Duration != 30*time.Minute {
		t.Errorf("Cache.TTL = %v, want %v", cfg.Cache.TTL.Duration, 30*time.Minute)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/kantar.toml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}
