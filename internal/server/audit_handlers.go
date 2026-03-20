package server

import (
	"net/http"
	"strconv"

	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	logs, err := s.deps.Queries.ListAuditLogs(r.Context(), sqlc.ListAuditLogsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

func (s *Server) handleVerifyAuditChain(w http.ResponseWriter, r *http.Request) {
	if s.deps.AuditLogger == nil {
		writeError(w, http.StatusServiceUnavailable, "audit logger not available")
		return
	}

	result, err := s.deps.AuditLogger.Verify(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "verification failed")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
