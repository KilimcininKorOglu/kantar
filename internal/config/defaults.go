package config

import (
	"fmt"
	"time"
)

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:    "0.0.0.0",
			Port:    8080,
			Workers: 0,
		},
		Storage: StorageConfig{
			Type: "filesystem",
			Path: "/var/lib/kantar/data",
		},
		Database: DatabaseConfig{
			Type: "postgres",
			Postgres: PostgresConfig{
				Host:    "localhost",
				Port:    5432,
				Name:    "kantar",
				User:    "kantar",
				SSLMode: "disable",
			},
		},
		Auth: AuthConfig{
			Type:       "local",
			SessionTTL: Duration{24 * time.Hour},
			TokenTTL:   Duration{90 * 24 * time.Hour},
		},
		Cache: CacheConfig{
			Enabled: true,
			Type:    "memory",
			MaxSize: "2GB",
			TTL:     Duration{1 * time.Hour},
			Redis: RedisConfig{
				Addr: "localhost:6379",
				DB:   0,
			},
		},
		Logging: LoggingConfig{
			Level:          "info",
			Format:         "json",
			AuditEnabled:   true,
			AuditPath:      "/var/lib/kantar/logs/audit.log",
			AuditRetention: Duration{365 * 24 * time.Hour},
		},
		Notifications: NotificationsConfig{
			Enabled: false,
		},
		Registries: map[string]RegistryConfig{},
	}
}

// UnmarshalText implements encoding.TextUnmarshaler for Duration.
// Supports Go duration strings (e.g., "24h", "30m") and day notation (e.g., "7d", "365d").
func (d *Duration) UnmarshalText(text []byte) error {
	s := string(text)

	// Handle day notation: "7d" → 7 * 24h
	if len(s) > 0 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			d.Duration = time.Duration(days) * 24 * time.Hour
			return nil
		}
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = dur
	return nil
}

// MarshalText implements encoding.TextMarshaler for Duration.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}
