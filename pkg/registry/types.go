package registry

import (
	"io"
	"net/http"
	"time"
)

// PackageMeta holds metadata about a package without the actual content.
type PackageMeta struct {
	Name          string         `json:"name"`
	Registry      EcosystemType  `json:"registry"`
	Description   string         `json:"description,omitempty"`
	License       string         `json:"license,omitempty"`
	Versions      []VersionInfo  `json:"versions,omitempty"`
	LatestVersion string         `json:"latestVersion,omitempty"`
	Maintainers   []string       `json:"maintainers,omitempty"`
	Homepage      string         `json:"homepage,omitempty"`
	Repository    string         `json:"repository,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	Extra         map[string]any `json:"extra,omitempty"`
}

// VersionInfo holds metadata for a specific package version.
type VersionInfo struct {
	Version    string    `json:"version"`
	Published  time.Time `json:"published"`
	Size       int64     `json:"size"`
	Checksum   string    `json:"checksum"`
	Deprecated bool      `json:"deprecated,omitempty"`
	Yanked     bool      `json:"yanked,omitempty"`
}

// Package represents a full package including its content.
type Package struct {
	Meta    PackageMeta `json:"meta"`
	Version string      `json:"version"`
	Content io.Reader   `json:"-"`
	Size    int64       `json:"size"`

	// Checksums for integrity verification.
	SHA256 string `json:"sha256,omitempty"`
	SHA1   string `json:"sha1,omitempty"`
	MD5    string `json:"md5,omitempty"`

	// Dependencies of this package.
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

// Dependency represents a package dependency.
type Dependency struct {
	Name         string `json:"name"`
	VersionRange string `json:"versionRange"`
	Optional     bool   `json:"optional,omitempty"`
	Dev          bool   `json:"dev,omitempty"`
}

// ValidationResult holds the result of package validation.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// Route defines an HTTP route that a plugin wants to register.
type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
}
