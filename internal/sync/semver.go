// Package sync provides recursive dependency resolution and sync for Kantar.
package sync

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// parseVersion parses a semver string like "1.2.3" into [major, minor, patch].
// Pre-release suffix (e.g., "-beta.1") is stripped for comparison but returned.
func parseVersion(v string) (major, minor, patch int, prerelease string, ok bool) {
	v = strings.TrimPrefix(v, "v")

	// Split off pre-release
	if idx := strings.IndexByte(v, '-'); idx >= 0 {
		prerelease = v[idx+1:]
		v = v[:idx]
	}

	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 3 {
		// Pad missing parts with 0
		for len(parts) < 3 {
			parts = append(parts, "0")
		}
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, "", false
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, "", false
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, "", false
	}

	return major, minor, patch, prerelease, true
}

// compareSemver compares two version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareSemver(a, b string) int {
	aMaj, aMin, aPat, aPre, aOk := parseVersion(a)
	bMaj, bMin, bPat, bPre, bOk := parseVersion(b)

	if !aOk || !bOk {
		return strings.Compare(a, b)
	}

	if aMaj != bMaj {
		return cmpInt(aMaj, bMaj)
	}
	if aMin != bMin {
		return cmpInt(aMin, bMin)
	}
	if aPat != bPat {
		return cmpInt(aPat, bPat)
	}

	// Pre-release versions have lower precedence than release
	if aPre != "" && bPre == "" {
		return -1
	}
	if aPre == "" && bPre != "" {
		return 1
	}

	return strings.Compare(aPre, bPre)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// matchesNpmRange checks if a concrete version satisfies an npm range string.
// Supports: exact, *, ^, ~, >=, <=, >, <, and || (OR).
func matchesNpmRange(version, rangeStr string) bool {
	rangeStr = strings.TrimSpace(rangeStr)

	if rangeStr == "" || rangeStr == "*" || rangeStr == "latest" {
		return true
	}

	// Handle || (OR) — any alternative must match
	if strings.Contains(rangeStr, "||") {
		for _, alt := range strings.Split(rangeStr, "||") {
			if matchesNpmRange(version, strings.TrimSpace(alt)) {
				return true
			}
		}
		return false
	}

	// Handle space-separated (AND) constraints
	if strings.Contains(rangeStr, " ") && !strings.HasPrefix(rangeStr, ">=") && !strings.HasPrefix(rangeStr, "<=") {
		parts := strings.Fields(rangeStr)
		for _, p := range parts {
			if !matchesSingleRange(version, p) {
				return false
			}
		}
		return true
	}

	return matchesSingleRange(version, rangeStr)
}

func matchesSingleRange(version, r string) bool {
	r = strings.TrimSpace(r)

	if r == "" || r == "*" {
		return true
	}

	vMaj, vMin, vPat, _, vOk := parseVersion(version)
	if !vOk {
		return false
	}

	switch {
	case strings.HasPrefix(r, "^"):
		return matchCaret(vMaj, vMin, vPat, r[1:])
	case strings.HasPrefix(r, "~"):
		return matchTilde(vMaj, vMin, vPat, r[1:])
	case strings.HasPrefix(r, ">="):
		return compareSemver(version, strings.TrimSpace(r[2:])) >= 0
	case strings.HasPrefix(r, "<="):
		return compareSemver(version, strings.TrimSpace(r[2:])) <= 0
	case strings.HasPrefix(r, ">"):
		return compareSemver(version, strings.TrimSpace(r[1:])) > 0
	case strings.HasPrefix(r, "<"):
		return compareSemver(version, strings.TrimSpace(r[1:])) < 0
	case strings.HasPrefix(r, "="):
		return compareSemver(version, strings.TrimSpace(r[1:])) == 0
	default:
		// Exact version match
		return compareSemver(version, r) == 0
	}
}

// matchCaret: ^1.2.3 means >=1.2.3 and <2.0.0 (same major)
// ^0.2.3 means >=0.2.3 and <0.3.0 (same major.minor when major is 0)
// ^0.0.3 means >=0.0.3 and <0.0.4 (exact when major.minor is 0.0)
func matchCaret(vMaj, vMin, vPat int, rangeVer string) bool {
	rMaj, rMin, rPat, _, ok := parseVersion(rangeVer)
	if !ok {
		return false
	}

	// Must be >= the specified version
	if compareSemver(fmt.Sprintf("%d.%d.%d", vMaj, vMin, vPat), fmt.Sprintf("%d.%d.%d", rMaj, rMin, rPat)) < 0 {
		return false
	}

	if rMaj != 0 {
		return vMaj == rMaj
	}
	if rMin != 0 {
		return vMaj == 0 && vMin == rMin
	}
	return vMaj == 0 && vMin == 0 && vPat == rPat
}

// matchTilde: ~1.2.3 means >=1.2.3 and <1.3.0 (same major.minor)
func matchTilde(vMaj, vMin, vPat int, rangeVer string) bool {
	rMaj, rMin, rPat, _, ok := parseVersion(rangeVer)
	if !ok {
		return false
	}

	if compareSemver(fmt.Sprintf("%d.%d.%d", vMaj, vMin, vPat), fmt.Sprintf("%d.%d.%d", rMaj, rMin, rPat)) < 0 {
		return false
	}

	return vMaj == rMaj && vMin == rMin
}

// selectBestVersion picks the highest version from candidates that satisfies rangeStr.
// Pre-release versions are skipped unless the range explicitly targets one.
func selectBestVersion(candidates []string, rangeStr string) (string, bool) {
	// Sort descending by semver
	sorted := make([]string, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool {
		return compareSemver(sorted[i], sorted[j]) > 0
	})

	rangeHasPrerelease := strings.Contains(rangeStr, "-")

	for _, v := range sorted {
		_, _, _, pre, ok := parseVersion(v)
		if !ok {
			continue
		}
		// Skip pre-release unless range explicitly targets one
		if pre != "" && !rangeHasPrerelease {
			continue
		}
		if matchesNpmRange(v, rangeStr) {
			return v, true
		}
	}

	return "", false
}
