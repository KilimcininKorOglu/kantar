package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

type registryResponse struct {
	ID               int64     `json:"id"`
	Ecosystem        string    `json:"ecosystem"`
	Mode             string    `json:"mode"`
	Upstream         string    `json:"upstream"`
	AutoSync         bool      `json:"autoSync"`
	AutoSyncInterval string    `json:"autoSyncInterval"`
	MaxVersions      int64     `json:"maxVersions"`
	Enabled          bool      `json:"enabled"`
	ConfigJSON       string    `json:"configJson,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func toRegistryResponse(r sqlc.Registry) registryResponse {
	return registryResponse{
		ID:               r.ID,
		Ecosystem:        r.Ecosystem,
		Mode:             r.Mode,
		Upstream:         r.Upstream,
		AutoSync:         r.AutoSync == 1,
		AutoSyncInterval: r.AutoSyncInterval,
		MaxVersions:      r.MaxVersions,
		Enabled:          r.Enabled == 1,
		ConfigJSON:       r.ConfigJson,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func (s *Server) handleListRegistries(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	registries, err := s.deps.Queries.ListRegistries(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list registries")
		return
	}

	resp := make([]registryResponse, len(registries))
	for i, reg := range registries {
		resp[i] = toRegistryResponse(reg)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetRegistry(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	ecosystem := chi.URLParam(r, "ecosystem")
	reg, err := s.deps.Queries.GetRegistry(r.Context(), ecosystem)
	if err != nil {
		writeError(w, http.StatusNotFound, "registry not found")
		return
	}

	writeJSON(w, http.StatusOK, toRegistryResponse(reg))
}

type updateRegistryRequest struct {
	Mode             string `json:"mode"`
	Upstream         string `json:"upstream"`
	AutoSync         *bool  `json:"autoSync"`
	AutoSyncInterval string `json:"autoSyncInterval"`
	MaxVersions      *int64 `json:"maxVersions"`
	Enabled          *bool  `json:"enabled"`
}

func (s *Server) handleUpdateRegistry(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	ecosystem := chi.URLParam(r, "ecosystem")
	var req updateRegistryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get existing to merge
	existing, err := s.deps.Queries.GetRegistry(r.Context(), ecosystem)
	if err != nil {
		writeError(w, http.StatusNotFound, "registry not found")
		return
	}

	mode := existing.Mode
	if req.Mode != "" {
		mode = req.Mode
	}
	upstream := existing.Upstream
	if req.Upstream != "" {
		upstream = req.Upstream
	}
	autoSync := existing.AutoSync
	if req.AutoSync != nil {
		if *req.AutoSync {
			autoSync = 1
		} else {
			autoSync = 0
		}
	}
	autoSyncInterval := existing.AutoSyncInterval
	if req.AutoSyncInterval != "" {
		autoSyncInterval = req.AutoSyncInterval
	}
	maxVersions := existing.MaxVersions
	if req.MaxVersions != nil {
		maxVersions = *req.MaxVersions
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		if *req.Enabled {
			enabled = 1
		} else {
			enabled = 0
		}
	}

	if err := s.deps.Queries.UpsertRegistry(r.Context(), sqlc.UpsertRegistryParams{
		Ecosystem:        ecosystem,
		Mode:             mode,
		Upstream:         upstream,
		AutoSync:         autoSync,
		AutoSyncInterval: autoSyncInterval,
		MaxVersions:      maxVersions,
		Enabled:          enabled,
		ConfigJson:       existing.ConfigJson,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update registry")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
