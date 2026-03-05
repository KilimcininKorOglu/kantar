package nuget

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func extractParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// normalizeID returns the package ID in lowercase form as required by NuGet V3
// flat-container endpoints.
func normalizeID(id string) string {
	return strings.ToLower(id)
}
