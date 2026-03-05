package helm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

func extractParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

// parseChartFilename extracts the chart name and version from a filename like "mychart-1.0.0.tgz".
// The version is identified as the last segment after a dash that starts with a digit.
// Returns empty strings if the filename cannot be parsed.
func parseChartFilename(filename string) (string, string) {
	// Remove .tgz suffix.
	if !strings.HasSuffix(filename, ".tgz") {
		return "", ""
	}
	base := strings.TrimSuffix(filename, ".tgz")
	if base == "" {
		return "", ""
	}

	// Find the last dash followed by a digit — that's where the version starts.
	// This handles chart names with dashes like "my-cool-chart-1.2.3".
	for i := len(base) - 1; i > 0; i-- {
		if base[i-1] == '-' && base[i] >= '0' && base[i] <= '9' {
			return base[:i-1], base[i:]
		}
	}

	return "", ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// generateIndexYAML builds a Helm-compatible index.yaml from stored chart metadata.
// It manually constructs the YAML string to avoid requiring a YAML library.
func generateIndexYAML(entries map[string][]chartMeta) string {
	var b strings.Builder

	b.WriteString("apiVersion: v1\n")
	b.WriteString("entries:\n")

	// Sort chart names for deterministic output.
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		versions := entries[name]

		// Sort versions descending by created time so latest comes first.
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].Created.After(versions[j].Created)
		})

		b.WriteString(fmt.Sprintf("  %s:\n", yamlEscape(name)))

		for _, v := range versions {
			b.WriteString("  - name: ")
			b.WriteString(yamlEscape(v.Name))
			b.WriteString("\n")

			b.WriteString("    version: ")
			b.WriteString(yamlEscape(v.Version))
			b.WriteString("\n")

			if v.Description != "" {
				b.WriteString("    description: ")
				b.WriteString(yamlEscape(v.Description))
				b.WriteString("\n")
			}

			if v.AppVersion != "" {
				b.WriteString("    appVersion: ")
				b.WriteString(yamlEscape(v.AppVersion))
				b.WriteString("\n")
			}

			b.WriteString("    digest: ")
			b.WriteString(v.Digest)
			b.WriteString("\n")

			b.WriteString("    created: ")
			b.WriteString(v.Created.UTC().Format("2006-01-02T15:04:05.000000000Z"))
			b.WriteString("\n")

			if len(v.URLs) > 0 {
				b.WriteString("    urls:\n")
				for _, u := range v.URLs {
					b.WriteString("    - ")
					b.WriteString(yamlEscape(u))
					b.WriteString("\n")
				}
			}
		}
	}

	if len(entries) == 0 {
		b.WriteString("  {}\n")
	}

	b.WriteString("generated: \"")
	b.WriteString("Kantar Helm Registry")
	b.WriteString("\"\n")

	return b.String()
}

// yamlEscape wraps a string in double quotes if it contains characters
// that could be misinterpreted in YAML. Simple strings are returned as-is.
func yamlEscape(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, ":#{}[]|>&*!%@`'\",\n\\") {
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		return `"` + escaped + `"`
	}
	return s
}
