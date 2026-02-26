package policy

import (
	"context"
	"testing"
	"time"
)

func TestLicensePolicyAllowed(t *testing.T) {
	p := &LicensePolicy{
		Allowed: []string{"MIT", "Apache-2.0"},
		Blocked: []string{"GPL-3.0"},
		Action:  ActionBlock,
	}

	pkg := &PackageInfo{Name: "test", License: "MIT"}
	violations := p.Evaluate(context.Background(), pkg)
	if len(violations) != 0 {
		t.Errorf("expected no violations for MIT, got %d", len(violations))
	}
}

func TestLicensePolicyBlocked(t *testing.T) {
	p := &LicensePolicy{
		Allowed: []string{"MIT"},
		Blocked: []string{"GPL-3.0"},
		Action:  ActionBlock,
	}

	pkg := &PackageInfo{Name: "test", License: "GPL-3.0"}
	violations := p.Evaluate(context.Background(), pkg)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Action != ActionBlock {
		t.Errorf("expected block action, got %s", violations[0].Action)
	}
}

func TestLicensePolicyNoLicense(t *testing.T) {
	p := &LicensePolicy{
		Allowed: []string{"MIT"},
		Action:  ActionWarn,
	}

	pkg := &PackageInfo{Name: "test", License: ""}
	violations := p.Evaluate(context.Background(), pkg)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for no license, got %d", len(violations))
	}
}

func TestSizePolicy(t *testing.T) {
	p := &SizePolicy{
		MaxPackageSize: 500 * 1024 * 1024, // 500MB
		MaxLayerCount:  20,
		Action:         ActionBlock,
	}

	// Within limits
	pkg := &PackageInfo{Name: "test", Size: 100 * 1024 * 1024}
	violations := p.Evaluate(context.Background(), pkg)
	if len(violations) != 0 {
		t.Errorf("expected no violations, got %d", len(violations))
	}

	// Over size limit
	pkg.Size = 600 * 1024 * 1024
	violations = p.Evaluate(context.Background(), pkg)
	if len(violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(violations))
	}

	// Over layer limit
	pkg.Size = 100
	pkg.LayerCount = 25
	violations = p.Evaluate(context.Background(), pkg)
	if len(violations) != 1 {
		t.Errorf("expected 1 violation for layers, got %d", len(violations))
	}
}

func TestAgePolicy(t *testing.T) {
	p := &AgePolicy{
		MinPackageAge:  7 * 24 * time.Hour,
		MinMaintainers: 2,
		Action:         ActionWarn,
	}

	// Too new
	pkg := &PackageInfo{
		Name:        "new-pkg",
		PublishedAt: time.Now().Add(-1 * 24 * time.Hour),
		Maintainers: 3,
	}
	violations := p.Evaluate(context.Background(), pkg)
	if len(violations) != 1 {
		t.Errorf("expected 1 violation for new package, got %d", len(violations))
	}

	// Old enough, enough maintainers
	pkg.PublishedAt = time.Now().Add(-30 * 24 * time.Hour)
	violations = p.Evaluate(context.Background(), pkg)
	if len(violations) != 0 {
		t.Errorf("expected no violations, got %d", len(violations))
	}

	// Too few maintainers
	pkg.Maintainers = 1
	violations = p.Evaluate(context.Background(), pkg)
	if len(violations) != 1 {
		t.Errorf("expected 1 violation for low maintainers, got %d", len(violations))
	}
}

func TestVersionPolicy(t *testing.T) {
	p := &VersionPolicy{
		AllowPreRelease: false,
		AllowDeprecated: false,
		Action:          ActionBlock,
	}

	// Pre-release blocked
	pkg := &PackageInfo{Name: "test", PreRelease: true}
	violations := p.Evaluate(context.Background(), pkg)
	if len(violations) != 1 {
		t.Errorf("expected 1 violation for pre-release, got %d", len(violations))
	}

	// Deprecated blocked
	pkg = &PackageInfo{Name: "test", Deprecated: true}
	violations = p.Evaluate(context.Background(), pkg)
	if len(violations) != 1 {
		t.Errorf("expected 1 violation for deprecated, got %d", len(violations))
	}

	// Allow pre-release
	p.AllowPreRelease = true
	pkg = &PackageInfo{Name: "test", PreRelease: true}
	violations = p.Evaluate(context.Background(), pkg)
	if len(violations) != 0 {
		t.Errorf("expected no violations when pre-release allowed, got %d", len(violations))
	}
}

func TestEngineEvaluate(t *testing.T) {
	engine := NewEngine()

	engine.Register(&LicensePolicy{
		Allowed: []string{"MIT"},
		Blocked: []string{"GPL-3.0"},
		Action:  ActionBlock,
	})

	engine.Register(&SizePolicy{
		MaxPackageSize: 100,
		Action:         ActionWarn,
	})

	// Pass all policies
	pkg := &PackageInfo{Name: "good-pkg", License: "MIT", Size: 50}
	result := engine.Evaluate(context.Background(), pkg)
	if !result.Allowed {
		t.Error("expected allowed for good package")
	}

	// Fail license policy
	pkg = &PackageInfo{Name: "bad-pkg", License: "GPL-3.0", Size: 50}
	result = engine.Evaluate(context.Background(), pkg)
	if result.Allowed {
		t.Error("expected blocked for GPL package")
	}
	if len(result.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(result.Violations))
	}

	// Warn on size
	pkg = &PackageInfo{Name: "big-pkg", License: "MIT", Size: 200}
	result = engine.Evaluate(context.Background(), pkg)
	if !result.Allowed {
		t.Error("expected allowed (size is warning, not block)")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
}
