package sync

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input               string
		major, minor, patch int
		pre                 string
		ok                  bool
	}{
		{"1.2.3", 1, 2, 3, "", true},
		{"0.0.1", 0, 0, 1, "", true},
		{"10.20.30", 10, 20, 30, "", true},
		{"1.2.3-beta.1", 1, 2, 3, "beta.1", true},
		{"v2.0.0", 2, 0, 0, "", true},
		{"1.0", 1, 0, 0, "", true},
		{"abc", 0, 0, 0, "", false},
	}

	for _, tt := range tests {
		maj, min, pat, pre, ok := parseVersion(tt.input)
		if ok != tt.ok {
			t.Errorf("parseVersion(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if !ok {
			continue
		}
		if maj != tt.major || min != tt.minor || pat != tt.patch || pre != tt.pre {
			t.Errorf("parseVersion(%q) = %d.%d.%d-%s, want %d.%d.%d-%s",
				tt.input, maj, min, pat, pre, tt.major, tt.minor, tt.patch, tt.pre)
		}
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.2.0", "1.1.9", 1},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0", "1.0.0-beta", 1},
	}

	for _, tt := range tests {
		got := compareSemver(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareSemver(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestMatchesNpmRange(t *testing.T) {
	tests := []struct {
		version  string
		rangeStr string
		want     bool
	}{
		// Exact
		{"1.2.3", "1.2.3", true},
		{"1.2.4", "1.2.3", false},

		// Star / empty
		{"5.0.0", "*", true},
		{"1.0.0", "", true},
		{"3.0.0", "latest", true},

		// Caret ^
		{"1.2.3", "^1.2.3", true},
		{"1.9.0", "^1.2.3", true},
		{"2.0.0", "^1.2.3", false},
		{"1.2.2", "^1.2.3", false},
		{"0.2.5", "^0.2.3", true},
		{"0.3.0", "^0.2.3", false},

		// Tilde ~
		{"1.2.5", "~1.2.3", true},
		{"1.3.0", "~1.2.3", false},
		{"1.2.2", "~1.2.3", false},

		// Comparison operators
		{"2.0.0", ">=1.0.0", true},
		{"0.9.0", ">=1.0.0", false},
		{"1.0.0", "<=1.0.0", true},
		{"1.0.1", "<=1.0.0", false},
		{"2.0.0", ">1.0.0", true},
		{"1.0.0", ">1.0.0", false},
		{"0.9.0", "<1.0.0", true},
		{"1.0.0", "<1.0.0", false},

		// OR (||)
		{"1.0.0", "^1.0.0 || ^2.0.0", true},
		{"2.5.0", "^1.0.0 || ^2.0.0", true},
		{"3.0.0", "^1.0.0 || ^2.0.0", false},
	}

	for _, tt := range tests {
		got := matchesNpmRange(tt.version, tt.rangeStr)
		if got != tt.want {
			t.Errorf("matchesNpmRange(%q, %q) = %v, want %v", tt.version, tt.rangeStr, got, tt.want)
		}
	}
}

func TestSelectBestVersion(t *testing.T) {
	candidates := []string{"4.17.0", "4.18.0", "4.18.2", "5.0.0-beta.1", "3.9.0"}

	tests := []struct {
		rangeStr string
		want     string
		ok       bool
	}{
		{"^4.17.0", "4.18.2", true},
		{"~4.18.0", "4.18.2", true},
		{"^3.0.0", "3.9.0", true},
		{"^5.0.0", "", false}, // 5.0.0-beta.1 skipped (pre-release)
		{">=4.18.0", "4.18.2", true},
		{"*", "4.18.2", true},
		{"4.17.0", "4.17.0", true}, // Exact match
	}

	for _, tt := range tests {
		got, ok := selectBestVersion(candidates, tt.rangeStr)
		if ok != tt.ok || got != tt.want {
			t.Errorf("selectBestVersion(candidates, %q) = (%q, %v), want (%q, %v)",
				tt.rangeStr, got, ok, tt.want, tt.ok)
		}
	}
}
