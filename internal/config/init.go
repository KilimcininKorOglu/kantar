package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultConfigTemplate = `# Kantar Configuration — Bootstrap Only
# https://github.com/KilimcininKorOglu/kantar
#
# This file contains only bootstrap settings required to start the server.
# Runtime settings (logging, cache, registries, policies) are managed via
# the database and Web UI after first startup.

[server]
host = "0.0.0.0"
port = 8080
# tls_cert = "/path/to/cert.pem"
# tls_key = "/path/to/key.pem"
# base_url = "https://kantar.company.internal"
workers = 0  # 0 = number of CPUs

[storage]
type = "filesystem"
path = "/var/lib/kantar/data"

# [storage.s3]
# endpoint = "https://minio.local:9000"
# bucket = "kantar"
# access_key = "${KANTAR_S3_ACCESS_KEY}"
# secret_key = "${KANTAR_S3_SECRET_KEY}"
# region = "us-east-1"

[database]
type = "postgres"

[database.postgres]
host = "localhost"
port = 5432
name = "kantar"
user = "kantar"
password = "${KANTAR_DB_PASSWORD}"
ssl_mode = "disable"

[auth]
type = "local"
# jwt_secret = ""  # Auto-generated if empty
`

const defaultPolicyTemplate = `# Kantar Security Policy

[policy.license]
allowed = ["MIT", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "ISC"]
blocked = ["GPL-3.0", "AGPL-3.0"]
action = "block"  # block | warn | log

[policy.vulnerability]
block_severity = ["critical", "high"]
warn_severity = ["medium"]
allow_severity = ["low"]
scanner = "none"  # grype | trivy | none
auto_scan = false
scan_on_sync = false

[policy.age]
min_package_age = "7d"
# min_maintainers = 2

[policy.size]
max_package_size = "500MB"
# max_layer_count = 20  # Docker specific

[policy.naming]
# blocked_scopes = ["@evil-corp"]
# blocked_prefixes = ["__test"]

[policy.version]
# pin_strategy = "minor"  # major | minor | patch | exact
allow_prerelease = false
allow_deprecated = false
`

// InitConfig creates default configuration files at the specified directory.
// If dir is empty, it uses the current working directory.
// Returns the paths of created files.
func InitConfig(dir string) ([]string, error) {
	if dir == "" {
		dir = "."
	}

	files := map[string]string{
		"kantar.toml":            defaultConfigTemplate,
		"policies/security.toml": defaultPolicyTemplate,
	}

	var created []string

	for relPath, content := range files {
		fullPath := filepath.Join(dir, relPath)

		if _, err := os.Stat(fullPath); err == nil {
			return created, fmt.Errorf("file already exists: %s (use --force to overwrite)", fullPath)
		}

		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return created, fmt.Errorf("creating directory for %s: %w", fullPath, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return created, fmt.Errorf("writing %s: %w", fullPath, err)
		}

		created = append(created, fullPath)
	}

	return created, nil
}

// InitConfigForce creates default configuration files, overwriting existing ones.
func InitConfigForce(dir string) ([]string, error) {
	if dir == "" {
		dir = "."
	}

	files := map[string]string{
		"kantar.toml":            defaultConfigTemplate,
		"policies/security.toml": defaultPolicyTemplate,
	}

	var created []string

	for relPath, content := range files {
		fullPath := filepath.Join(dir, relPath)

		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return created, fmt.Errorf("creating directory for %s: %w", fullPath, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return created, fmt.Errorf("writing %s: %w", fullPath, err)
		}

		created = append(created, fullPath)
	}

	return created, nil
}
