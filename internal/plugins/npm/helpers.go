package npm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

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
