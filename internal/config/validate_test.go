package config

import (
	"testing"
	"time"
)

func TestValidateDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if err := Validate(cfg); err != nil {
		t.Errorf("default config should be valid, got: %v", err)
	}
}

func TestValidateInvalidPort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Port = 99999
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for invalid port")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Errors) == 0 {
		t.Error("expected at least one validation error")
	}
}

func TestValidateTLSMismatch(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.TLSCert = "/path/to/cert.pem"
	// TLSKey intentionally empty
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for TLS key without cert")
	}
}

func TestValidatePostgresRequiredFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Database.Type = "postgres"
	cfg.Database.Postgres.Host = ""
	cfg.Database.Postgres.Name = ""
	cfg.Database.Postgres.User = ""
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for missing postgres fields")
	}
	ve := err.(*ValidationError)
	if len(ve.Errors) < 3 {
		t.Errorf("expected at least 3 errors, got %d", len(ve.Errors))
	}
}

func TestValidateS3RequiredFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.Type = "s3"
	// All S3 fields empty
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for missing S3 fields")
	}
	ve := err.(*ValidationError)
	if len(ve.Errors) < 4 {
		t.Errorf("expected at least 4 S3 errors, got %d", len(ve.Errors))
	}
}

func TestValidateInvalidDatabaseType(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Database.Type = "mysql"
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for unsupported database type")
	}
}

func TestValidateInvalidAuthType(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.Type = "kerberos"
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for unsupported auth type")
	}
}

func TestValidateLDAPRequiredFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.Type = "ldap"
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for missing LDAP fields")
	}
}

func TestValidateInvalidSessionTTL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.SessionTTL = Duration{0}
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for zero session TTL")
	}
}

func TestValidateRedisRequiredFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Cache.Type = "redis"
	cfg.Cache.Redis.Addr = ""
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for missing Redis addr")
	}
}

func TestValidateDisabledCacheSkipsValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Cache.Enabled = false
	cfg.Cache.Type = "invalid"
	if err := Validate(cfg); err != nil {
		t.Errorf("disabled cache should skip validation, got: %v", err)
	}
}

func TestValidateInvalidLogLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logging.Level = "verbose"
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for invalid log level")
	}
}

func TestValidateValidPostgresConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Database.Type = "postgres"
	cfg.Database.Postgres.Host = "localhost"
	cfg.Database.Postgres.Port = 5432
	cfg.Database.Postgres.Name = "kantar"
	cfg.Database.Postgres.User = "kantar"
	cfg.Database.Postgres.Password = "secret"
	cfg.Auth.SessionTTL = Duration{24 * time.Hour}
	cfg.Auth.TokenTTL = Duration{90 * 24 * time.Hour}
	if err := Validate(cfg); err != nil {
		t.Errorf("valid postgres config should pass, got: %v", err)
	}
}
