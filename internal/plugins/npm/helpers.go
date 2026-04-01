package npm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/httpclient"
)

var httpClient = httpclient.Shared

func extractParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func bytesReader(data []byte) io.Reader {
	return bytes.NewReader(data)
}

func decodeBase64(data string) []byte {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil
	}
	return decoded
}

// findBestVersion picks the highest semver from candidates matching an npm range.
func findBestVersion(candidates []string, rangeStr string) string {
	rangeStr = strings.TrimSpace(rangeStr)
	if rangeStr == "" || rangeStr == "*" || rangeStr == "latest" {
		// Return highest stable
		sorted := sortVersionsDesc(candidates)
		for _, v := range sorted {
			if !strings.Contains(v, "-") {
				return v
			}
		}
		if len(sorted) > 0 {
			return sorted[0]
		}
		return ""
	}

	// Check exact match first
	for _, c := range candidates {
		if c == rangeStr {
			return c
		}
	}

	sorted := sortVersionsDesc(candidates)
	for _, v := range sorted {
		if strings.Contains(v, "-") {
			continue // skip pre-release
		}
		if npmRangeMatch(v, rangeStr) {
			return v
		}
	}
	return ""
}

func sortVersionsDesc(versions []string) []string {
	sorted := make([]string, len(versions))
	copy(sorted, versions)
	sort.Slice(sorted, func(i, j int) bool {
		return semverCompare(sorted[i], sorted[j]) > 0
	})
	return sorted
}

func semverCompare(a, b string) int {
	ap := parseSemver(a)
	bp := parseSemver(b)
	for i := 0; i < 3; i++ {
		if ap[i] != bp[i] {
			if ap[i] > bp[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.IndexByte(v, '-'); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		result[i], _ = strconv.Atoi(parts[i])
	}
	return result
}

func npmRangeMatch(version, rangeStr string) bool {
	rangeStr = strings.TrimSpace(rangeStr)
	if rangeStr == "" || rangeStr == "*" {
		return true
	}

	// Handle ||
	if strings.Contains(rangeStr, "||") {
		for _, alt := range strings.Split(rangeStr, "||") {
			if npmRangeMatch(version, strings.TrimSpace(alt)) {
				return true
			}
		}
		return false
	}

	v := parseSemver(version)

	switch {
	case strings.HasPrefix(rangeStr, "^"):
		r := parseSemver(rangeStr[1:])
		if v[0] < r[0] || (v[0] == r[0] && v[1] < r[1]) || (v[0] == r[0] && v[1] == r[1] && v[2] < r[2]) {
			return false
		}
		if r[0] != 0 {
			return v[0] == r[0]
		}
		if r[1] != 0 {
			return v[0] == 0 && v[1] == r[1]
		}
		return v[0] == 0 && v[1] == 0 && v[2] == r[2]

	case strings.HasPrefix(rangeStr, "~"):
		r := parseSemver(rangeStr[1:])
		if v[0] < r[0] || (v[0] == r[0] && v[1] < r[1]) || (v[0] == r[0] && v[1] == r[1] && v[2] < r[2]) {
			return false
		}
		return v[0] == r[0] && v[1] == r[1]

	case strings.HasPrefix(rangeStr, ">="):
		return semverCompare(version, strings.TrimSpace(rangeStr[2:])) >= 0
	case strings.HasPrefix(rangeStr, "<="):
		return semverCompare(version, strings.TrimSpace(rangeStr[2:])) <= 0
	case strings.HasPrefix(rangeStr, ">"):
		return semverCompare(version, strings.TrimSpace(rangeStr[1:])) > 0
	case strings.HasPrefix(rangeStr, "<"):
		return semverCompare(version, strings.TrimSpace(rangeStr[1:])) < 0

	default:
		return semverCompare(version, rangeStr) == 0
	}
}

// Ensure fmt is used
var _ = fmt.Sprintf
