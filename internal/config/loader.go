package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// configSearchPaths defines the default locations to look for config files.
var configSearchPaths = []string{
	"./kantar.toml",
	"/etc/kantar/kantar.toml",
}

// Load reads and parses the Kantar configuration from file.
// It searches in the following order:
// 1. Explicit path (if provided)
// 2. KANTAR_CONFIG environment variable
// 3. Default search paths (./kantar.toml, /etc/kantar/kantar.toml)
//
// Environment variables in the format ${VAR} or ${VAR:-default} are
// interpolated before parsing.
func Load(path string) (*Config, error) {
	configPath, err := resolveConfigPath(path)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	interpolated := interpolateEnvVars(string(raw))

	cfg := DefaultConfig()
	if _, err := toml.Decode(interpolated, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", configPath, err)
	}

	return cfg, nil
}

// resolveConfigPath determines which config file to use.
func resolveConfigPath(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("config file not found: %s", explicit)
		}
		return explicit, nil
	}

	if envPath := os.Getenv("KANTAR_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err != nil {
			return "", fmt.Errorf("config file from KANTAR_CONFIG not found: %s", envPath)
		}
		return envPath, nil
	}

	for _, p := range configSearchPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("no config file found; searched: %s (set KANTAR_CONFIG or use --config)",
		strings.Join(configSearchPaths, ", "))
}

// envVarPattern matches ${VAR} and ${VAR:-default} patterns.
var envVarPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)(?::-(.*?))?\}`)

// interpolateEnvVars replaces ${VAR} and ${VAR:-default} with environment variable values.
func interpolateEnvVars(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		submatches := envVarPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		varName := submatches[1]
		defaultVal := ""
		hasDefault := len(submatches) >= 3 && submatches[2] != ""

		if hasDefault {
			defaultVal = submatches[2]
		}

		if val, ok := os.LookupEnv(varName); ok {
			return val
		}

		if hasDefault {
			return defaultVal
		}

		return match
	})
}
