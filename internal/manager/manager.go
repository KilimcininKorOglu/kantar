// Package manager provides the package lifecycle management core.
package manager

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"time"

	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

// PackageStatus represents the approval status of a package.
type PackageStatus string

const (
	StatusPending  PackageStatus = "pending"
	StatusApproved PackageStatus = "approved"
	StatusBlocked  PackageStatus = "blocked"
)

// Manager handles package lifecycle operations.
type Manager struct {
	queries *sqlc.Queries
}

// New creates a new package Manager.
func New(db *sql.DB) *Manager {
	return &Manager{
		queries: sqlc.New(db),
	}
}

// GetPackage retrieves a package by registry type and name.
func (m *Manager) GetPackage(ctx context.Context, registryType, name string) (*sqlc.Package, error) {
	pkg, err := m.queries.GetPackage(ctx, sqlc.GetPackageParams{
		RegistryType: registryType,
		Name:         name,
	})
	if err != nil {
		return nil, fmt.Errorf("getting package %s/%s: %w", registryType, name, err)
	}
	return &pkg, nil
}

// ListPackages lists packages for a registry type with pagination.
func (m *Manager) ListPackages(ctx context.Context, registryType string, limit, offset int64) ([]sqlc.Package, error) {
	return m.queries.ListPackages(ctx, sqlc.ListPackagesParams{
		RegistryType: registryType,
		Limit:        limit,
		Offset:       offset,
	})
}

// ListPackagesByStatus lists packages filtered by status.
func (m *Manager) ListPackagesByStatus(ctx context.Context, registryType string, status PackageStatus, limit, offset int64) ([]sqlc.Package, error) {
	return m.queries.ListPackagesByStatus(ctx, sqlc.ListPackagesByStatusParams{
		RegistryType: registryType,
		Status:       string(status),
		Limit:        limit,
		Offset:       offset,
	})
}

// SearchPackages searches packages by name pattern.
func (m *Manager) SearchPackages(ctx context.Context, registryType, query string, limit, offset int64) ([]sqlc.Package, error) {
	return m.queries.SearchPackages(ctx, sqlc.SearchPackagesParams{
		RegistryType: registryType,
		Name:         "%" + query + "%",
		Limit:        limit,
		Offset:       offset,
	})
}

// RequestPackage creates a new package request (status = pending).
func (m *Manager) RequestPackage(ctx context.Context, registryType, name, requestedBy string) (*sqlc.Package, error) {
	pkg, err := m.queries.CreatePackage(ctx, sqlc.CreatePackageParams{
		RegistryType: registryType,
		Name:         name,
		Status:       string(StatusPending),
		RequestedBy:  requestedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("creating package request %s/%s: %w", registryType, name, err)
	}
	return &pkg, nil
}

// ApprovePackage approves a pending package.
func (m *Manager) ApprovePackage(ctx context.Context, packageID int64, approvedBy string) error {
	return m.queries.UpdatePackageStatus(ctx, sqlc.UpdatePackageStatusParams{
		Status:        string(StatusApproved),
		ApprovedBy:    approvedBy,
		BlockedReason: "",
		ID:            packageID,
	})
}

// BlockPackage blocks a package with a reason.
func (m *Manager) BlockPackage(ctx context.Context, packageID int64, reason string) error {
	return m.queries.UpdatePackageStatus(ctx, sqlc.UpdatePackageStatusParams{
		Status:        string(StatusBlocked),
		ApprovedBy:    "",
		BlockedReason: reason,
		ID:            packageID,
	})
}

// AddVersion adds a version to a package.
func (m *Manager) AddVersion(ctx context.Context, packageID int64, version string, size int64, sha256, storagePath string) (*sqlc.PackageVersion, error) {
	v, err := m.queries.CreatePackageVersion(ctx, sqlc.CreatePackageVersionParams{
		PackageID:      packageID,
		Version:        version,
		Size:           size,
		ChecksumSha256: sha256,
		StoragePath:    storagePath,
		SyncedAt:       sql.NullTime{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("adding version %s: %w", version, err)
	}
	return &v, nil
}

// IsAllowed checks if a package/version is allowed under the given mode.
func IsAllowed(mode string, allowlist []AllowRule, blocklist []string, name, version string) bool {
	switch mode {
	case "allowlist":
		for _, rule := range allowlist {
			matched, _ := path.Match(rule.Name, name)
			if !matched {
				continue
			}
			for _, vp := range rule.Versions {
				if vp == "*" {
					return true
				}
				vMatched, _ := path.Match(vp, version)
				if vMatched {
					return true
				}
			}
		}
		return false

	case "mirror":
		for _, blocked := range blocklist {
			matched, _ := path.Match(blocked, name)
			if matched {
				return false
			}
		}
		return true

	default:
		return false
	}
}

// AllowRule defines a package allowlist entry.
type AllowRule struct {
	Name     string
	Versions []string
}
