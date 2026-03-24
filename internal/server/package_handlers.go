package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/audit"
	"github.com/KilimcininKorOglu/kantar/internal/auth"
	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

type packageResponse struct {
	ID            int64     `json:"id"`
	RegistryType  string    `json:"registryType"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	License       string    `json:"license"`
	Homepage      string    `json:"homepage"`
	Repository    string    `json:"repository"`
	Status        string    `json:"status"`
	RequestedBy   string    `json:"requestedBy"`
	ApprovedBy    string    `json:"approvedBy"`
	BlockedReason string    `json:"blockedReason"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func toPackageResponse(p sqlc.Package) packageResponse {
	return packageResponse{
		ID:            p.ID,
		RegistryType:  p.RegistryType,
		Name:          p.Name,
		Description:   p.Description,
		License:       p.License,
		Homepage:      p.Homepage,
		Repository:    p.Repository,
		Status:        p.Status,
		RequestedBy:   p.RequestedBy,
		ApprovedBy:    p.ApprovedBy,
		BlockedReason: p.BlockedReason,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func (s *Server) handleListPackages(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	registry := r.URL.Query().Get("registry")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if registry == "" {
		registry = "npm"
	}

	var pkgs []sqlc.Package
	var err error

	switch {
	case search != "":
		pkgs, err = s.deps.Queries.SearchPackages(r.Context(), sqlc.SearchPackagesParams{
			RegistryType: registry,
			Name:         "%" + search + "%",
			Limit:        limit,
			Offset:       offset,
		})
	case status != "":
		pkgs, err = s.deps.Queries.ListPackagesByStatus(r.Context(), sqlc.ListPackagesByStatusParams{
			RegistryType: registry,
			Status:       status,
			Limit:        limit,
			Offset:       offset,
		})
	default:
		pkgs, err = s.deps.Queries.ListPackages(r.Context(), sqlc.ListPackagesParams{
			RegistryType: registry,
			Limit:        limit,
			Offset:       offset,
		})
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list packages")
		return
	}

	resp := make([]packageResponse, len(pkgs))
	for i, p := range pkgs {
		resp[i] = toPackageResponse(p)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetPackageByName(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	registry := chi.URLParam(r, "registry")
	name := chi.URLParam(r, "name")

	pkg, err := s.deps.Queries.GetPackage(r.Context(), sqlc.GetPackageParams{
		RegistryType: registry,
		Name:         name,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "package not found")
		return
	}

	writeJSON(w, http.StatusOK, toPackageResponse(pkg))
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

	writeJSON(w, http.StatusOK, toPackageResponse(pkg))
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
