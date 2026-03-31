// Package registry defines the public plugin interface for Kantar ecosystem plugins.
// Each package ecosystem (Docker, npm, PyPI, Go Modules, Cargo, Maven, NuGet, Helm)
// implements this interface to integrate with the Kantar core engine.
package registry

import (
	"context"
	"net/http"
)

// EcosystemType identifies a package ecosystem.
type EcosystemType string

const (
	EcosystemDocker EcosystemType = "docker"
	EcosystemNPM    EcosystemType = "npm"
	EcosystemPyPI   EcosystemType = "pypi"
	EcosystemGoMod  EcosystemType = "gomod"
	EcosystemCargo  EcosystemType = "cargo"
	EcosystemMaven  EcosystemType = "maven"
	EcosystemNuGet  EcosystemType = "nuget"
	EcosystemHelm   EcosystemType = "helm"
)

// RegistryPlugin is the core interface that every ecosystem plugin must implement.
type RegistryPlugin interface {
	// Name returns the human-readable plugin name.
	Name() string

	// Ecosystem returns the ecosystem type this plugin handles.
	Ecosystem() EcosystemType

	// Search queries the upstream registry for packages matching the query string.
	Search(ctx context.Context, query string) ([]PackageMeta, error)

	// FetchPackage downloads a specific package version from upstream.
	FetchPackage(ctx context.Context, name, version string) (*Package, error)

	// FetchMetadata retrieves metadata for a package from upstream without downloading it.
	FetchMetadata(ctx context.Context, name string) (*PackageMeta, error)

	// ServePackage handles native protocol requests (e.g., Docker Registry API v2, npm registry API).
	ServePackage(w http.ResponseWriter, r *http.Request)

	// PublishPackage stores a locally published package.
	PublishPackage(ctx context.Context, pkg *Package) error

	// DeletePackage removes a specific version of a package.
	DeletePackage(ctx context.Context, name, version string) error

	// ValidatePackage runs ecosystem-specific validation on a package.
	ValidatePackage(ctx context.Context, pkg *Package) (*ValidationResult, error)

	// Configure applies plugin-specific configuration.
	Configure(config map[string]any) error

	// DefaultConfig returns the default configuration values for this plugin.
	DefaultConfig() map[string]any

	// Routes returns the HTTP routes this plugin needs mounted.
	Routes() []Route
}
