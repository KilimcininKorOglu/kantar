// Package config handles loading, parsing, and validating Kantar configuration.
package config

import "time"

// Config is the root configuration structure for Kantar.
type Config struct {
	Server        ServerConfig                  `toml:"server"`
	Storage       StorageConfig                 `toml:"storage"`
	Database      DatabaseConfig                `toml:"database"`
	Auth          AuthConfig                    `toml:"auth"`
	Cache         CacheConfig                   `toml:"cache"`
	Logging       LoggingConfig                 `toml:"logging"`
	Notifications NotificationsConfig           `toml:"notifications"`
	Registries    map[string]RegistryConfig      `toml:"registry"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
	TLSCert string `toml:"tls_cert"`
	TLSKey  string `toml:"tls_key"`
	BaseURL string `toml:"base_url"`
	Workers int    `toml:"workers"`
}

// StorageConfig holds package storage settings.
type StorageConfig struct {
	Type string        `toml:"type"`
	Path string        `toml:"path"`
	S3   S3Config      `toml:"s3"`
}

// S3Config holds S3-compatible storage settings.
type S3Config struct {
	Endpoint  string `toml:"endpoint"`
	Bucket    string `toml:"bucket"`
	AccessKey string `toml:"access_key"`
	SecretKey string `toml:"secret_key"`
	Region    string `toml:"region"`
}

// DatabaseConfig holds database settings.
type DatabaseConfig struct {
	Type     string         `toml:"type"`
	Postgres PostgresConfig `toml:"postgres"`
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Name     string `toml:"name"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	SSLMode  string `toml:"ssl_mode"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Type       string        `toml:"type"`
	SessionTTL Duration      `toml:"session_ttl"`
	TokenTTL   Duration      `toml:"token_ttl"`
	JWTSecret  string        `toml:"jwt_secret"`
	LDAP       LDAPConfig    `toml:"ldap"`
	OIDC       OIDCConfig    `toml:"oidc"`
}

// LDAPConfig holds LDAP authentication settings.
type LDAPConfig struct {
	URL          string `toml:"url"`
	BaseDN       string `toml:"base_dn"`
	BindDN       string `toml:"bind_dn"`
	BindPassword string `toml:"bind_password"`
	UserFilter   string `toml:"user_filter"`
	GroupFilter  string `toml:"group_filter"`
}

// OIDCConfig holds OIDC authentication settings.
type OIDCConfig struct {
	Issuer       string   `toml:"issuer"`
	ClientID     string   `toml:"client_id"`
	ClientSecret string   `toml:"client_secret"`
	Scopes       []string `toml:"scopes"`
}

// CacheConfig holds caching settings.
type CacheConfig struct {
	Enabled bool      `toml:"enabled"`
	Type    string    `toml:"type"`
	MaxSize string    `toml:"max_size"`
	TTL     Duration  `toml:"ttl"`
	Redis   RedisConfig `toml:"redis"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `toml:"addr"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level          string   `toml:"level"`
	Format         string   `toml:"format"`
	AuditEnabled   bool     `toml:"audit_enabled"`
	AuditPath      string   `toml:"audit_path"`
	AuditRetention Duration `toml:"audit_retention"`
}

// NotificationsConfig holds notification settings.
type NotificationsConfig struct {
	Enabled bool            `toml:"enabled"`
	Webhook WebhookConfig   `toml:"webhook"`
	Email   EmailConfig     `toml:"email"`
}

// WebhookConfig holds webhook notification settings.
type WebhookConfig struct {
	URL    string   `toml:"url"`
	Events []string `toml:"events"`
}

// EmailConfig holds email notification settings.
type EmailConfig struct {
	SMTPHost string   `toml:"smtp_host"`
	SMTPPort int      `toml:"smtp_port"`
	From     string   `toml:"from"`
	To       []string `toml:"to"`
	Events   []string `toml:"events"`
}

// RegistryConfig holds per-ecosystem registry configuration.
type RegistryConfig struct {
	Mode             string          `toml:"mode"`
	Upstream         string          `toml:"upstream"`
	AutoSync         bool            `toml:"auto_sync"`
	AutoSyncInterval Duration        `toml:"auto_sync_interval"`
	MaxVersions      int             `toml:"max_versions"`
	Enabled          bool            `toml:"enabled"`
	Allowlist        AllowlistConfig `toml:"allowlist"`
	Blocklist        []string        `toml:"blocklist"`
}

// AllowlistConfig holds the allowlist for a registry.
type AllowlistConfig struct {
	Packages []PackageRule `toml:"packages"`
	Images   []ImageRule   `toml:"images"`
}

// PackageRule defines an allowlist entry for a package.
type PackageRule struct {
	Name     string   `toml:"name"`
	Versions []string `toml:"versions"`
}

// ImageRule defines an allowlist entry for a Docker image.
type ImageRule struct {
	Name string   `toml:"name"`
	Tags []string `toml:"tags"`
}

// Duration wraps time.Duration to support TOML string parsing (e.g., "24h", "7d").
type Duration struct {
	time.Duration
}
