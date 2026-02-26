// Package policy implements the declarative policy engine for Kantar.
package policy

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Action defines what happens when a policy is violated.
type Action string

const (
	ActionBlock Action = "block"
	ActionWarn  Action = "warn"
	ActionLog   Action = "log"
	ActionAllow Action = "allow"
)

// Violation describes a policy violation.
type Violation struct {
	Policy  string `json:"policy"`
	Message string `json:"message"`
	Action  Action `json:"action"`
}

// Result holds the result of policy evaluation.
type Result struct {
	Allowed    bool        `json:"allowed"`
	Violations []Violation `json:"violations,omitempty"`
	Warnings   []Violation `json:"warnings,omitempty"`
}

// PackageInfo holds the package metadata needed for policy evaluation.
type PackageInfo struct {
	Name         string
	Version      string
	License      string
	Size         int64
	PublishedAt  time.Time
	Maintainers  int
	Deprecated   bool
	PreRelease   bool
	LayerCount   int // Docker specific
}

// Policy is the interface for individual policy checks.
type Policy interface {
	Name() string
	Evaluate(ctx context.Context, pkg *PackageInfo) []Violation
}

// Engine runs all registered policies against a package.
type Engine struct {
	policies []Policy
}

// NewEngine creates a new policy engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Register adds a policy to the engine.
func (e *Engine) Register(p Policy) {
	e.policies = append(e.policies, p)
}

// Evaluate runs all policies against the given package.
// The most restrictive action wins (block > warn > log > allow).
func (e *Engine) Evaluate(ctx context.Context, pkg *PackageInfo) *Result {
	result := &Result{Allowed: true}

	for _, p := range e.policies {
		violations := p.Evaluate(ctx, pkg)
		for _, v := range violations {
			switch v.Action {
			case ActionBlock:
				result.Allowed = false
				result.Violations = append(result.Violations, v)
			case ActionWarn:
				result.Warnings = append(result.Warnings, v)
			case ActionLog:
				result.Warnings = append(result.Warnings, v)
			}
		}
	}

	return result
}

// --- License Policy ---

// LicensePolicy checks package licenses against allowed/blocked lists.
type LicensePolicy struct {
	Allowed []string
	Blocked []string
	Action  Action
}

func (p *LicensePolicy) Name() string { return "license" }

func (p *LicensePolicy) Evaluate(_ context.Context, pkg *PackageInfo) []Violation {
	if pkg.License == "" {
		return []Violation{{
			Policy:  "license",
			Message: "package has no license specified",
			Action:  p.Action,
		}}
	}

	license := strings.ToUpper(pkg.License)

	for _, blocked := range p.Blocked {
		if strings.ToUpper(blocked) == license {
			return []Violation{{
				Policy:  "license",
				Message: fmt.Sprintf("license %q is blocked", pkg.License),
				Action:  p.Action,
			}}
		}
	}

	if len(p.Allowed) > 0 {
		found := false
		for _, allowed := range p.Allowed {
			if strings.ToUpper(allowed) == license {
				found = true
				break
			}
		}
		if !found {
			return []Violation{{
				Policy:  "license",
				Message: fmt.Sprintf("license %q is not in the allowed list", pkg.License),
				Action:  p.Action,
			}}
		}
	}

	return nil
}

// --- Size Policy ---

// SizePolicy checks package size against limits.
type SizePolicy struct {
	MaxPackageSize int64
	MaxLayerCount  int
	Action         Action
}

func (p *SizePolicy) Name() string { return "size" }

func (p *SizePolicy) Evaluate(_ context.Context, pkg *PackageInfo) []Violation {
	var violations []Violation

	if p.MaxPackageSize > 0 && pkg.Size > p.MaxPackageSize {
		violations = append(violations, Violation{
			Policy:  "size",
			Message: fmt.Sprintf("package size %d bytes exceeds limit %d bytes", pkg.Size, p.MaxPackageSize),
			Action:  p.Action,
		})
	}

	if p.MaxLayerCount > 0 && pkg.LayerCount > p.MaxLayerCount {
		violations = append(violations, Violation{
			Policy:  "size",
			Message: fmt.Sprintf("layer count %d exceeds limit %d", pkg.LayerCount, p.MaxLayerCount),
			Action:  p.Action,
		})
	}

	return violations
}

// --- Age Policy ---

// AgePolicy checks package publish date against minimum age.
type AgePolicy struct {
	MinPackageAge  time.Duration
	MinMaintainers int
	Action         Action
}

func (p *AgePolicy) Name() string { return "age" }

func (p *AgePolicy) Evaluate(_ context.Context, pkg *PackageInfo) []Violation {
	var violations []Violation

	if p.MinPackageAge > 0 && !pkg.PublishedAt.IsZero() {
		age := time.Since(pkg.PublishedAt)
		if age < p.MinPackageAge {
			violations = append(violations, Violation{
				Policy:  "age",
				Message: fmt.Sprintf("package age %s is less than minimum %s", age.Round(time.Hour), p.MinPackageAge),
				Action:  p.Action,
			})
		}
	}

	if p.MinMaintainers > 0 && pkg.Maintainers < p.MinMaintainers {
		violations = append(violations, Violation{
			Policy:  "age",
			Message: fmt.Sprintf("package has %d maintainers, minimum is %d", pkg.Maintainers, p.MinMaintainers),
			Action:  p.Action,
		})
	}

	return violations
}

// --- Version Policy ---

// VersionPolicy checks version constraints.
type VersionPolicy struct {
	AllowPreRelease bool
	AllowDeprecated bool
	Action          Action
}

func (p *VersionPolicy) Name() string { return "version" }

func (p *VersionPolicy) Evaluate(_ context.Context, pkg *PackageInfo) []Violation {
	var violations []Violation

	if !p.AllowPreRelease && pkg.PreRelease {
		violations = append(violations, Violation{
			Policy:  "version",
			Message: "pre-release versions are not allowed",
			Action:  p.Action,
		})
	}

	if !p.AllowDeprecated && pkg.Deprecated {
		violations = append(violations, Violation{
			Policy:  "version",
			Message: "deprecated packages are not allowed",
			Action:  p.Action,
		})
	}

	return violations
}
