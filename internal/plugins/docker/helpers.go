package docker

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// extractPathParam extracts a URL path parameter using chi.
func extractPathParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

// writeError writes a Docker Registry API v2 error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]string{
			{"code": code, "message": message},
		},
	})
}

// computeDigest computes the sha256 digest of data in Docker format.
func computeDigest(data []byte) string {
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}

// generateUUID generates a simple UUID for upload tracking.
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// detectManifestMediaType tries to detect the media type from manifest JSON.
func detectManifestMediaType(data []byte) string {
	var manifest struct {
		MediaType     string `json:"mediaType"`
		SchemaVersion int    `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &manifest); err == nil {
		if manifest.MediaType != "" {
			return manifest.MediaType
		}
		if manifest.SchemaVersion == 2 {
			return "application/vnd.docker.distribution.manifest.v2+json"
		}
	}

	// Check if it looks like an OCI manifest
	if strings.Contains(string(data), "application/vnd.oci") {
		return "application/vnd.oci.image.manifest.v1+json"
	}

	return "application/vnd.docker.distribution.manifest.v2+json"
}
