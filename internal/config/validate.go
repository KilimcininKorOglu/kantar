package config

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidationError collects multiple validation errors.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation failed:\n  - %s", strings.Join(e.Errors, "\n  - "))
}

func (e *ValidationError) add(format string, args ...any) {
	e.Errors = append(e.Errors, fmt.Sprintf(format, args...))
}

func (e *ValidationError) hasErrors() bool {
	return len(e.Errors) > 0
}

// Validate checks the configuration for logical consistency and required fields.
func Validate(cfg *Config) error {
	ve := &ValidationError{}

	validateServer(cfg, ve)
	validateStorage(cfg, ve)
	validateDatabase(cfg, ve)
	validateAuth(cfg, ve)
	validateCache(cfg, ve)
	validateLogging(cfg, ve)

	if ve.hasErrors() {
		return ve
	}
	return nil
}

func validateServer(cfg *Config, ve *ValidationError) {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		ve.add("server.port must be between 1 and 65535, got %d", cfg.Server.Port)
	}

	if cfg.Server.TLSCert != "" && cfg.Server.TLSKey == "" {
		ve.add("server.tls_key is required when server.tls_cert is set")
	}
	if cfg.Server.TLSKey != "" && cfg.Server.TLSCert == "" {
		ve.add("server.tls_cert is required when server.tls_key is set")
	}

	if cfg.Server.BaseURL != "" {
		if _, err := url.Parse(cfg.Server.BaseURL); err != nil {
			ve.add("server.base_url is not a valid URL: %s", cfg.Server.BaseURL)
		}
	}

	if cfg.Server.Workers < 0 {
		ve.add("server.workers must be >= 0, got %d", cfg.Server.Workers)
	}
}

func validateStorage(cfg *Config, ve *ValidationError) {
	switch cfg.Storage.Type {
	case "filesystem":
		if cfg.Storage.Path == "" {
			ve.add("storage.path is required when storage.type is 'filesystem'")
		}
	case "s3":
		if cfg.Storage.S3.Endpoint == "" {
			ve.add("storage.s3.endpoint is required when storage.type is 's3'")
		}
		if cfg.Storage.S3.Bucket == "" {
			ve.add("storage.s3.bucket is required when storage.type is 's3'")
		}
		if cfg.Storage.S3.AccessKey == "" {
			ve.add("storage.s3.access_key is required when storage.type is 's3'")
		}
		if cfg.Storage.S3.SecretKey == "" {
			ve.add("storage.s3.secret_key is required when storage.type is 's3'")
		}
	default:
		ve.add("storage.type must be 'filesystem' or 's3', got %q", cfg.Storage.Type)
	}
}

func validateDatabase(cfg *Config, ve *ValidationError) {
	if cfg.Database.Type != "postgres" {
		ve.add("database.type must be 'postgres', got %q", cfg.Database.Type)
		return
	}

	pg := cfg.Database.Postgres
	if pg.Host == "" {
		ve.add("database.postgres.host is required")
	}
	if pg.Port < 1 || pg.Port > 65535 {
		ve.add("database.postgres.port must be between 1 and 65535, got %d", pg.Port)
	}
	if pg.Name == "" {
		ve.add("database.postgres.name is required")
	}
	if pg.User == "" {
		ve.add("database.postgres.user is required")
	}
}

func validateAuth(cfg *Config, ve *ValidationError) {
	validTypes := map[string]bool{"local": true, "ldap": true, "oidc": true}
	if !validTypes[cfg.Auth.Type] {
		ve.add("auth.type must be 'local', 'ldap', or 'oidc', got %q", cfg.Auth.Type)
	}

	if cfg.Auth.Type == "ldap" {
		if cfg.Auth.LDAP.URL == "" {
			ve.add("auth.ldap.url is required when auth.type is 'ldap'")
		}
		if cfg.Auth.LDAP.BaseDN == "" {
			ve.add("auth.ldap.base_dn is required when auth.type is 'ldap'")
		}
	}

	if cfg.Auth.Type == "oidc" {
		if cfg.Auth.OIDC.Issuer == "" {
			ve.add("auth.oidc.issuer is required when auth.type is 'oidc'")
		}
		if cfg.Auth.OIDC.ClientID == "" {
			ve.add("auth.oidc.client_id is required when auth.type is 'oidc'")
		}
	}

	if cfg.Auth.SessionTTL.Duration <= 0 {
		ve.add("auth.session_ttl must be positive")
	}
	if cfg.Auth.TokenTTL.Duration <= 0 {
		ve.add("auth.token_ttl must be positive")
	}
}

func validateCache(cfg *Config, ve *ValidationError) {
	if !cfg.Cache.Enabled {
		return
	}

	switch cfg.Cache.Type {
	case "memory":
		// No additional requirements
	case "redis":
		if cfg.Cache.Redis.Addr == "" {
			ve.add("cache.redis.addr is required when cache.type is 'redis'")
		}
	default:
		ve.add("cache.type must be 'memory' or 'redis', got %q", cfg.Cache.Type)
	}
}

func validateLogging(cfg *Config, ve *ValidationError) {
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[cfg.Logging.Level] {
		ve.add("logging.level must be 'debug', 'info', 'warn', or 'error', got %q", cfg.Logging.Level)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[cfg.Logging.Format] {
		ve.add("logging.format must be 'json' or 'text', got %q", cfg.Logging.Format)
	}

	if cfg.Logging.AuditEnabled && cfg.Logging.AuditPath == "" {
		ve.add("logging.audit_path is required when logging.audit_enabled is true")
	}
}
