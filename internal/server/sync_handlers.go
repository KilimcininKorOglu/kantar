package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

type syncJobResponse struct {
	ID             int64      `json:"id"`
	RegistryType   string     `json:"registryType"`
	PackageName    string     `json:"packageName"`
	Status         string     `json:"status"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	ErrorMessage   string     `json:"errorMessage,omitempty"`
	PackagesSynced int64      `json:"packagesSynced"`
	CreatedAt      time.Time  `json:"createdAt"`
}

func toSyncJobResponse(j sqlc.SyncJob) syncJobResponse {
	resp := syncJobResponse{
		ID:             j.ID,
		RegistryType:   j.RegistryType,
		PackageName:    j.PackageName,
		Status:         j.Status,
		ErrorMessage:   j.ErrorMessage,
		PackagesSynced: j.PackagesSynced,
		CreatedAt:      j.CreatedAt,
	}
	if j.StartedAt.Valid {
		resp.StartedAt = &j.StartedAt.Time
	}
	if j.CompletedAt.Valid {
		resp.CompletedAt = &j.CompletedAt.Time
	}
	return resp
}

func (s *Server) handleGetSyncJob(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := s.deps.Queries.GetSyncJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "sync job not found")
		return
	}

	writeJSON(w, http.StatusOK, toSyncJobResponse(job))
}

func (s *Server) handleListSyncJobs(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	jobs, err := s.deps.Queries.ListSyncJobs(r.Context(), sqlc.ListSyncJobsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sync jobs")
		return
	}

	resp := make([]syncJobResponse, len(jobs))
	for i, j := range jobs {
		resp[i] = toSyncJobResponse(j)
	}

	writeJSON(w, http.StatusOK, resp)
}
