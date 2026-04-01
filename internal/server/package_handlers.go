package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/audit"
	"github.com/KilimcininKorOglu/kantar/internal/auth"
	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
	syncp "github.com/KilimcininKorOglu/kantar/internal/sync"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

const packageCacheTTL = 60 * time.Second

func (s *Server) cacheGet(ctx context.Context, key string) []byte {
	if s.deps.Cache == nil {
		return nil
	}
	data, _ := s.deps.Cache.Get(ctx, key)
	return data
}

func (s *Server) cacheSet(ctx context.Context, key string, v any, ttl time.Duration) {
	if s.deps.Cache == nil {
		return
	}
	data, err := json.Marshal(v)
	if err == nil {
		s.deps.Cache.Set(ctx, key, data, ttl)
	}
}

func (s *Server) cacheInvalidatePackage(ctx context.Context, id int64, registryType, name string) {
	if s.deps.Cache == nil {
		return
	}
	s.deps.Cache.Delete(ctx, fmt.Sprintf("pkg:id:%d", id))
	s.deps.Cache.Delete(ctx, fmt.Sprintf("pkg:name:%s:%s", registryType, name))
}

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

	reg := chi.URLParam(r, "registry")
	name, _ := url.PathUnescape(chi.URLParam(r, "name"))

	cacheKey := fmt.Sprintf("pkg:name:%s:%s", reg, name)
	if cached := s.cacheGet(r.Context(), cacheKey); cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached)
		return
	}

	pkg, err := s.deps.Queries.GetPackage(r.Context(), sqlc.GetPackageParams{
		RegistryType: reg,
		Name:         name,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "package not found")
		return
	}

	resp := toPackageResponse(pkg)
	s.cacheSet(r.Context(), cacheKey, resp, packageCacheTTL)
	writeJSON(w, http.StatusOK, resp)
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

	cacheKey := fmt.Sprintf("pkg:id:%d", id)
	if cached := s.cacheGet(r.Context(), cacheKey); cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached)
		return
	}

	pkg, err := s.deps.Queries.GetPackageByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "package not found")
		return
	}

	resp := toPackageResponse(pkg)
	s.cacheSet(r.Context(), cacheKey, resp, packageCacheTTL)
	writeJSON(w, http.StatusOK, resp)
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

	// Invalidate cached package data
	if pkg, pkgErr := s.deps.Queries.GetPackageByID(r.Context(), id); pkgErr == nil {
		s.cacheInvalidatePackage(r.Context(), id, pkg.RegistryType, pkg.Name)
	}

	if s.deps.AuditLogger != nil {
		_ = s.deps.AuditLogger.Log(r.Context(), &audit.Event{
			EventType: audit.EventPackageApprove,
			Actor:     audit.Actor{Username: approvedBy, IP: r.RemoteAddr},
			Result:    "success",
		})
	}

	// Enqueue recursive dependency sync if engine is available
	if s.deps.SyncEngine != nil {
		pkg, pkgErr := s.deps.Queries.GetPackageByID(r.Context(), id)
		if pkgErr == nil {
			jobID, syncErr := s.deps.SyncEngine.Enqueue(r.Context(), &syncp.Job{
				PackageID:   id,
				PackageName: pkg.Name,
				Ecosystem:   registry.EcosystemType(pkg.RegistryType),
				ApprovedBy:  approvedBy,
				Options:     syncp.SyncOptions{MaxDepth: 10},
			})
			if syncErr == nil {
				writeJSON(w, http.StatusOK, map[string]any{"status": "approved", "syncJobId": jobID})
				return
			}
			s.logger.Warn("failed to enqueue sync job", "error", syncErr)
		}
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

	// Invalidate cached package data
	if pkg, pkgErr := s.deps.Queries.GetPackageByID(r.Context(), id); pkgErr == nil {
		s.cacheInvalidatePackage(r.Context(), id, pkg.RegistryType, pkg.Name)
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
