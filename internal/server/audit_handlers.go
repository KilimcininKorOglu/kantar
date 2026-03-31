package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

type auditLogResponse struct {
	ID               int64     `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	Event            string    `json:"event"`
	ActorUsername    string    `json:"actorUsername"`
	ActorIP          string    `json:"actorIp"`
	ActorUserAgent   string    `json:"actorUserAgent"`
	ResourceRegistry string    `json:"resourceRegistry"`
	ResourcePackage  string    `json:"resourcePackage"`
	ResourceVersion  string    `json:"resourceVersion"`
	Result           string    `json:"result"`
	MetadataJSON     string    `json:"metadataJson"`
	PrevHash         string    `json:"prevHash"`
	Hash             string    `json:"hash"`
}

func toAuditLogResponse(a sqlc.AuditLog) auditLogResponse {
	return auditLogResponse{
		ID:               a.ID,
		Timestamp:        a.Timestamp,
		Event:            a.Event,
		ActorUsername:    a.ActorUsername,
		ActorIP:          a.ActorIp,
		ActorUserAgent:   a.ActorUserAgent,
		ResourceRegistry: a.ResourceRegistry,
		ResourcePackage:  a.ResourcePackage,
		ResourceVersion:  a.ResourceVersion,
		Result:           a.Result,
		MetadataJSON:     a.MetadataJson,
		PrevHash:         a.PrevHash,
		Hash:             a.Hash,
	}
}

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

	resp := make([]auditLogResponse, len(logs))
	for i, l := range logs {
		resp[i] = toAuditLogResponse(l)
	}

	writeJSON(w, http.StatusOK, resp)
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
