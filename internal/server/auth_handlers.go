package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/KilimcininKorOglu/kantar/internal/audit"
	"github.com/KilimcininKorOglu/kantar/internal/auth"
	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	ExpiresAt time.Time    `json:"expiresAt"`
	User      userResponse `json:"user"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil || s.deps.JWTManager == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}

	user, err := s.deps.Queries.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if user.Active != 1 {
		writeError(w, http.StatusForbidden, "account disabled")
		return
	}

	if !auth.VerifyPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, expiresAt, err := s.deps.JWTManager.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		s.logger.Error("failed to generate token", "error", err)
		writeError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	if s.deps.AuditLogger != nil {
		_ = s.deps.AuditLogger.Log(r.Context(), &audit.Event{
			EventType: audit.EventUserLogin,
			Actor:     audit.Actor{Username: user.Username, IP: r.RemoteAddr},
			Result:    "success",
		})
	}

	setAuthCookies(w, r, token, expiresAt)

	writeJSON(w, http.StatusOK, loginResponse{
		ExpiresAt: expiresAt,
		User:      toUserResponse(user),
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	Timezone  string    `json:"timezone"`
	Locale    string    `json:"locale"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func toUserResponse(u sqlc.User) userResponse {
	resp := userResponse{
		ID:        u.ID,
		Username:  u.Username,
		Role:      u.Role,
		Active:    u.Active == 1,
		Timezone:  u.Timezone,
		Locale:    u.Locale,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	if u.Email.Valid {
		resp.Email = u.Email.String
	}
	return resp
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "service not ready")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "password hashing failed")
		return
	}

	email := sql.NullString{}
	if req.Email != "" {
		email = sql.NullString{String: req.Email, Valid: true}
	}

	user, err := s.deps.Queries.CreateUser(r.Context(), sqlc.CreateUserParams{
		Username:     req.Username,
		Email:        email,
		PasswordHash: hash,
		Role:         string(auth.RoleViewer),
	})
	if err != nil {
		writeError(w, http.StatusConflict, "username already exists")
		return
	}

	writeJSON(w, http.StatusCreated, toUserResponse(user))
}
