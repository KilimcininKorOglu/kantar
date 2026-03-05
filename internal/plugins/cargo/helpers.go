package cargo

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// extractParam extracts a URL path parameter using chi.
func extractParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// computePrefix computes the Cargo sparse index prefix for a crate name.
//
// Rules (RFC 2789):
//   - 1-character name  -> "1/"
//   - 2-character name  -> "2/"
//   - 3-character name  -> "3/{first_char}/"
//   - 4+ character name -> "{first_two}/{next_two}/"
func computePrefix(name string) string {
	lower := strings.ToLower(name)

	switch {
	case len(lower) == 1:
		return "1"
	case len(lower) == 2:
		return "2"
	case len(lower) == 3:
		return "3/" + lower[:1]
	default:
		return lower[:2] + "/" + lower[2:4]
	}
}
