package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/audit"
	"github.com/KilimcininKorOglu/kantar/internal/auth"
	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
	"github.com/KilimcininKorOglu/kantar/internal/policy"
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

	reg := r.URL.Query().Get("registry")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	perPage, _ := strconv.ParseInt(r.URL.Query().Get("perPage"), 10, 64)
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 || perPage > 100 {
		perPage = 50
	}
	offset := (page - 1) * perPage
	if reg == "" {
		reg = "npm"
	}

	var pkgs []sqlc.Package
	var err error

	switch {
	case search != "":
		pkgs, err = s.deps.Queries.SearchPackages(r.Context(), sqlc.SearchPackagesParams{
			RegistryType: reg,
			Name:         "%" + search + "%",
			Limit:        perPage,
			Offset:       offset,
		})
	case status != "":
		pkgs, err = s.deps.Queries.ListPackagesByStatus(r.Context(), sqlc.ListPackagesByStatusParams{
			RegistryType: reg,
			Status:       status,
			Limit:        perPage,
			Offset:       offset,
		})
	default:
		pkgs, err = s.deps.Queries.ListPackages(r.Context(), sqlc.ListPackagesParams{
			RegistryType: reg,
			Limit:        perPage,
			Offset:       offset,
		})
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list packages")
		return
	}

	total, _ := s.deps.Queries.CountPackages(r.Context(), reg)

	resp := make([]packageResponse, len(pkgs))
	for i, p := range pkgs {
		resp[i] = toPackageResponse(p)
	}

	writeJSON(w, http.StatusOK, paginatedResponse{
		Data:    resp,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
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

	// Evaluate policies before approval
	pkg, err := s.deps.Queries.GetPackageByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "package not found")
		return
	}

	// Fetch latest version for policy evaluation and sync job
	versions, _ := s.deps.Queries.ListPackageVersions(r.Context(), sqlc.ListPackageVersionsParams{
		PackageID: pkg.ID, Limit: 1, Offset: 0,
	})
	var latestVersion string
	if len(versions) > 0 {
		latestVersion = versions[0].Version
	}

	// Evaluate policies before approval
	dbPolicies, _ := s.deps.Queries.ListPolicies(r.Context())
	if len(dbPolicies) > 0 {
		policyEngine := policy.BuildFromDB(dbPolicies)
		pkgInfo := &policy.PackageInfo{
			Name:    pkg.Name,
			License: pkg.License,
		}
		if len(versions) > 0 {
			v := versions[0]
			pkgInfo.Version = v.Version
			pkgInfo.Size = v.Size
			pkgInfo.Deprecated = v.Deprecated == 1
			pkgInfo.PreRelease = strings.Contains(v.Version, "-")
			pkgInfo.PublishedAt = v.CreatedAt
		}

		result := policyEngine.Evaluate(r.Context(), pkgInfo)
		if !result.Allowed {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error":      "policy violation",
				"violations": result.Violations,
			})
			return
		}
	}

	if err := s.deps.Manager.ApprovePackage(r.Context(), id, approvedBy); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to approve package")
		return
	}

	// Invalidate cached package data
	{
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
		jobID, syncErr := s.deps.SyncEngine.Enqueue(r.Context(), &syncp.Job{
			PackageID:   id,
			PackageName: pkg.Name,
			Version:     latestVersion,
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

type packageVersionResponse struct {
	ID             int64     `json:"id"`
	PackageID      int64     `json:"packageId"`
	Version        string    `json:"version"`
	Size           int64     `json:"size"`
	ChecksumSha256 string    `json:"checksumSha256"`
	StoragePath    string    `json:"storagePath"`
	Deprecated     bool      `json:"deprecated"`
	Yanked         bool      `json:"yanked"`
	SyncedAt       string    `json:"syncedAt,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (s *Server) handleListPackageVersions(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid package id")
		return
	}

	versions, err := s.deps.Queries.ListPackageVersions(r.Context(), sqlc.ListPackageVersionsParams{
		PackageID: id, Limit: 100, Offset: 0,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list versions")
		return
	}

	resp := make([]packageVersionResponse, len(versions))
	for i, v := range versions {
		syncedAt := ""
		if v.SyncedAt.Valid {
			syncedAt = v.SyncedAt.Time.Format(time.RFC3339)
		}
		resp[i] = packageVersionResponse{
			ID:             v.ID,
			PackageID:      v.PackageID,
			Version:        v.Version,
			Size:           v.Size,
			ChecksumSha256: v.ChecksumSha256,
			StoragePath:    v.StoragePath,
			Deprecated:     v.Deprecated == 1,
			Yanked:         v.Yanked == 1,
			SyncedAt:       syncedAt,
			CreatedAt:      v.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
