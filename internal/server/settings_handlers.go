package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

type settingResponse struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func toSettingResponse(s sqlc.Setting) settingResponse {
	return settingResponse{
		Key:         s.Key,
		Value:       s.Value,
		Category:    s.Category,
		Description: s.Description,
		UpdatedAt:   s.UpdatedAt,
	}
}

func (s *Server) handleListSettings(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	settings, err := s.deps.Queries.ListSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list settings")
		return
	}

	resp := make([]settingResponse, len(settings))
	for i, st := range settings {
		resp[i] = toSettingResponse(st)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetSetting(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	key := chi.URLParam(r, "key")
	setting, err := s.deps.Queries.GetSetting(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusNotFound, "setting not found")
		return
	}

	writeJSON(w, http.StatusOK, toSettingResponse(setting))
}

type updateSettingRequest struct {
	Value string `json:"value"`
}

func (s *Server) handleUpdateSetting(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	key := chi.URLParam(r, "key")
	var req updateSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.deps.Queries.UpdateSettingValue(r.Context(), sqlc.UpdateSettingValueParams{
		Value: req.Value,
		Key:   key,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update setting")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

type bulkSettingUpdate struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s *Server) handleBulkUpdateSettings(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	var updates []bulkSettingUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	for _, u := range updates {
		s.deps.Queries.UpdateSettingValue(r.Context(), sqlc.UpdateSettingValueParams{
			Value: u.Value,
			Key:   u.Key,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "count": len(updates)})
}
