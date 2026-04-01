package server

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

type paginatedResponse struct {
	Data    any   `json:"data"`
	Total   int64 `json:"total"`
	Page    int64 `json:"page"`
	PerPage int64 `json:"perPage"`
}
