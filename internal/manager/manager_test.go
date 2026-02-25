package manager

import (
	"context"
	"testing"

	"github.com/KilimcininKorOglu/kantar/internal/database"
)

func TestPackageCRUD(t *testing.T) {
	db := database.NewTestDB(t)
	mgr := New(db.Conn())
	ctx := context.Background()

	// Request a package
	pkg, err := mgr.RequestPackage(ctx, "npm", "express", "dev-team")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if pkg.Status != "pending" {
		t.Errorf("expected pending, got %s", pkg.Status)
	}

	// Approve the package
	if err := mgr.ApprovePackage(ctx, pkg.ID, "admin"); err != nil {
		t.Fatalf("approve: %v", err)
	}

	// Get and verify
	got, err := mgr.GetPackage(ctx, "npm", "express")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "approved" {
		t.Errorf("expected approved, got %s", got.Status)
	}

	// Add version
	ver, err := mgr.AddVersion(ctx, pkg.ID, "4.18.2", 215000, "sha256abc", "npm/express/4.18.2.tgz")
	if err != nil {
		t.Fatalf("add version: %v", err)
	}
	if ver.Version != "4.18.2" {
		t.Errorf("expected 4.18.2, got %s", ver.Version)
	}
}

func TestBlockPackage(t *testing.T) {
	db := database.NewTestDB(t)
	mgr := New(db.Conn())
	ctx := context.Background()

	pkg, _ := mgr.RequestPackage(ctx, "npm", "malicious-pkg", "attacker")
	mgr.BlockPackage(ctx, pkg.ID, "supply-chain risk")

	got, _ := mgr.GetPackage(ctx, "npm", "malicious-pkg")
	if got.Status != "blocked" {
		t.Errorf("expected blocked, got %s", got.Status)
	}
	if got.BlockedReason != "supply-chain risk" {
		t.Errorf("expected reason, got %s", got.BlockedReason)
	}
}

func TestListAndSearch(t *testing.T) {
	db := database.NewTestDB(t)
	mgr := New(db.Conn())
	ctx := context.Background()

	mgr.RequestPackage(ctx, "npm", "express", "user")
	mgr.RequestPackage(ctx, "npm", "lodash", "user")
	mgr.RequestPackage(ctx, "npm", "react", "user")

	// List all
	pkgs, err := mgr.ListPackages(ctx, "npm", 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pkgs) != 3 {
		t.Errorf("expected 3, got %d", len(pkgs))
	}

	// Search
	results, err := mgr.SearchPackages(ctx, "npm", "exp", 10, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 search result, got %d", len(results))
	}
}

func TestIsAllowed(t *testing.T) {
	allowlist := []AllowRule{
		{Name: "express", Versions: []string{"4.*"}},
		{Name: "@types/*", Versions: []string{"*"}},
		{Name: "lodash", Versions: []string{"4.17.*"}},
	}

	tests := []struct {
		name    string
		mode    string
		pkg     string
		version string
		want    bool
	}{
		{"allowlist hit", "allowlist", "express", "4.18.2", true},
		{"allowlist miss version", "allowlist", "express", "5.0.0", false},
		{"allowlist miss pkg", "allowlist", "axios", "1.0.0", false},
		{"allowlist glob", "allowlist", "@types/node", "20.0.0", true},
		{"mirror allow", "mirror", "anything", "1.0.0", true},
		{"mirror blocked", "mirror", "evil-pkg", "1.0.0", false},
	}

	blocklist := []string{"evil-pkg"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAllowed(tt.mode, allowlist, blocklist, tt.pkg, tt.version)
			if got != tt.want {
				t.Errorf("IsAllowed(%s, %s, %s) = %v, want %v", tt.mode, tt.pkg, tt.version, got, tt.want)
			}
		})
	}
}
