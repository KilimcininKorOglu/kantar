package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/audit"
	"github.com/KilimcininKorOglu/kantar/internal/auth"
	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

func (s *Server) handleListPackages(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	registry := r.URL.Query().Get("registry")
	status := r.URL.Query().Get("status")
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	if status != "" && registry != "" {
		pkgs, err := s.deps.Queries.ListPackagesByStatus(r.Context(), sqlc.ListPackagesByStatusParams{
			RegistryType: registry,
			Status:       status,
			Limit:        limit,
			Offset:       offset,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list packages")
			return
		}
		writeJSON(w, http.StatusOK, pkgs)
		return
	}

	if registry == "" {
		registry = "npm"
	}

	pkgs, err := s.deps.Queries.ListPackages(r.Context(), sqlc.ListPackagesParams{
		RegistryType: registry,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list packages")
		return
	}
	writeJSON(w, http.StatusOK, pkgs)
}

func (s *Server) handleGetPackage(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid package id")
		return
	}

	pkg, err := s.deps.Queries.GetPackageByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "package not found")
		return
	}

	writeJSON(w, http.StatusOK, pkg)
}

func (s *Server) handleApprovePackage(w http.ResponseWriter, r *http.Request) {
	if s.deps.Manager == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid package id")
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	approvedBy := ""
	if claims != nil {
		approvedBy = claims.Username
	}

	if err := s.deps.Manager.ApprovePackage(r.Context(), id, approvedBy); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to approve package")
		return
	}

	if s.deps.AuditLogger != nil {
		_ = s.deps.AuditLogger.Log(r.Context(), &audit.Event{
			EventType: audit.EventPackageApprove,
			Actor:     audit.Actor{Username: approvedBy, IP: r.RemoteAddr},
			Result:    "success",
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

type blockRequest struct {
	Reason string `json:"reason"`
}

func (s *Server) handleBlockPackage(w http.ResponseWriter, r *http.Request) {
	if s.deps.Manager == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid package id")
		return
	}

	var req blockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.deps.Manager.BlockPackage(r.Context(), id, req.Reason); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to block package")
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	if s.deps.AuditLogger != nil && claims != nil {
		_ = s.deps.AuditLogger.Log(r.Context(), &audit.Event{
			EventType: audit.EventPackageBlock,
			Actor:     audit.Actor{Username: claims.Username, IP: r.RemoteAddr},
			Result:    "success",
			Metadata:  map[string]any{"reason": req.Reason},
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "blocked"})
}
