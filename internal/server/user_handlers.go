package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	users, err := s.deps.Queries.ListUsers(r.Context(), sqlc.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	resp := make([]userResponse, len(users))
	for i, u := range users {
		resp[i] = toUserResponse(u)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := s.deps.Queries.GetUserByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

type updateUserRequest struct {
	Email  string `json:"email"`
	Role   string `json:"role"`
	Active *bool  `json:"active"`
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := s.deps.Queries.GetUserByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	email := existing.Email
	if req.Email != "" {
		email = sql.NullString{String: req.Email, Valid: true}
	}

	role := existing.Role
	if req.Role != "" {
		role = req.Role
	}

	active := existing.Active
	if req.Active != nil {
		if *req.Active {
			active = 1
		} else {
			active = 0
		}
	}

	if err := s.deps.Queries.UpdateUser(r.Context(), sqlc.UpdateUserParams{
		Email:  email,
		Role:   role,
		Active: active,
		ID:     id,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := s.deps.Queries.DeleteUser(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
