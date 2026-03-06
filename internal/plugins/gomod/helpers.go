// Package gomod implements the Go Module Proxy plugin for Kantar.
package gomod

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
)

func extractParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

// encodePath encodes a Go module path per the GOPROXY spec: uppercase letters
// are replaced with an exclamation mark followed by the lowercase letter.
// For example, "github.com/Azure" becomes "github.com/!azure".
func encodePath(path string) string {
	var b strings.Builder
	for _, r := range path {
		if unicode.IsUpper(r) {
			b.WriteByte('!')
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// decodePath decodes a GOPROXY-encoded module path back to its original form.
// An exclamation mark followed by a lowercase letter is decoded to the
// corresponding uppercase letter.
func decodePath(encoded string) (string, bool) {
	var b strings.Builder
	escaped := false
	for _, r := range encoded {
		if escaped {
			if !unicode.IsLower(r) {
				return "", false
			}
			b.WriteRune(unicode.ToUpper(r))
			escaped = false
			continue
		}
		if r == '!' {
			escaped = true
			continue
		}
		b.WriteRune(r)
	}
	if escaped {
		return "", false
	}
	return b.String(), true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeText(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, text)
}

func bytesReader(data []byte) io.Reader {
	return bytes.NewReader(data)
}
