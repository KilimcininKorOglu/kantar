package pypi

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
)

var nonAlphanumericRe = regexp.MustCompile(`[-_.]+`)

func extractParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprint(w, body)
}

// normalizePkgName normalizes a package name according to PEP 503.
// All runs of underscores, hyphens, and periods are replaced with a single
// hyphen, and the result is lowercased.
func normalizePkgName(name string) string {
	return strings.ToLower(nonAlphanumericRe.ReplaceAllString(name, "-"))
}
