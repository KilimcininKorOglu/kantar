package server

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/KilimcininKorOglu/kantar/internal/auth"
	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := s.deps.Queries.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

type updateProfileRequest struct {
	Email    string `json:"email"`
	Timezone string `json:"timezone"`
	Locale   string `json:"locale"`
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := s.deps.Queries.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	email := existing.Email
	if req.Email != "" {
		email = sql.NullString{String: req.Email, Valid: true}
	}
	timezone := existing.Timezone
	if req.Timezone != "" {
		timezone = req.Timezone
	}
	locale := existing.Locale
	if req.Locale != "" {
		locale = req.Locale
	}

	if err := s.deps.Queries.UpdateUserProfile(r.Context(), sqlc.UpdateUserProfileParams{
		Email:    email,
		Timezone: timezone,
		Locale:   locale,
		ID:       claims.UserID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "new password must be at least 8 characters")
		return
	}

	user, err := s.deps.Queries.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if !auth.VerifyPassword(user.PasswordHash, req.CurrentPassword) {
		writeError(w, http.StatusForbidden, "current password is incorrect")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := s.deps.Queries.UpdateUserPassword(r.Context(), sqlc.UpdateUserPasswordParams{
		PasswordHash: newHash,
		ID:           claims.UserID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to change password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "password changed"})
}
