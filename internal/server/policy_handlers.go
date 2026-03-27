package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

type policyResponse struct {
	ID         int64          `json:"id"`
	Name       string         `json:"name"`
	PolicyType string         `json:"policyType"`
	Config     map[string]any `json:"config"`
	Enabled    bool           `json:"enabled"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

func toPolicyResponse(p sqlc.Policy) policyResponse {
	resp := policyResponse{
		ID:         p.ID,
		Name:       p.Name,
		PolicyType: p.PolicyType,
		Enabled:    p.Enabled == 1,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}

	// Parse TOML config into map for frontend
	var config map[string]any
	if _, err := toml.Decode(p.ConfigToml, &config); err == nil {
		resp.Config = config
	} else {
		resp.Config = map[string]any{"raw": p.ConfigToml}
	}

	return resp
}

func (s *Server) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	policies, err := s.deps.Queries.ListPolicies(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list policies")
		return
	}

	resp := make([]policyResponse, len(policies))
	for i, p := range policies {
		resp[i] = toPolicyResponse(p)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	name := chi.URLParam(r, "name")
	policy, err := s.deps.Queries.GetPolicy(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}

	writeJSON(w, http.StatusOK, toPolicyResponse(policy))
}

type updatePolicyRequest struct {
	Config  map[string]any `json:"config"`
	Enabled *bool          `json:"enabled"`
}

func (s *Server) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	name := chi.URLParam(r, "name")
	var req updatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := s.deps.Queries.GetPolicy(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}

	enabled := existing.Enabled
	if req.Enabled != nil {
		if *req.Enabled {
			enabled = 1
		} else {
			enabled = 0
		}
	}

	configToml := existing.ConfigToml
	if req.Config != nil {
		// Convert JSON config back to TOML
		var buf []byte
		buf, err = tomlMarshal(req.Config)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid config format")
			return
		}
		configToml = string(buf)
	}

	if err := s.deps.Queries.UpsertPolicy(r.Context(), sqlc.UpsertPolicyParams{
		Name:       name,
		PolicyType: existing.PolicyType,
		ConfigToml: configToml,
		Enabled:    enabled,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update policy")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleTogglePolicy(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	name := chi.URLParam(r, "name")
	existing, err := s.deps.Queries.GetPolicy(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}

	newEnabled := int64(1)
	if existing.Enabled == 1 {
		newEnabled = 0
	}

	if err := s.deps.Queries.UpdatePolicyEnabled(r.Context(), sqlc.UpdatePolicyEnabledParams{
		Enabled: newEnabled,
		Name:    name,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle policy")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "toggled", "enabled": newEnabled == 1})
}

// tomlMarshal converts a map to TOML string using BurntSushi/toml encoder.
func tomlMarshal(data map[string]any) ([]byte, error) {
	var buf []byte
	w := &tomlWriter{buf: &buf}
	enc := toml.NewEncoder(w)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	return buf, nil
}

type tomlWriter struct {
	buf *[]byte
}

func (w *tomlWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
