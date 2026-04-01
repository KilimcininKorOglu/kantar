package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// newStructuredLogger returns a chi middleware that logs requests using slog.
func newStructuredLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				status := ww.Status()
				level := slog.LevelInfo
				if status >= 500 {
					level = slog.LevelError
				} else if status >= 400 {
					level = slog.LevelWarn
				}

				logger.Log(r.Context(), level, "http request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", status,
					"bytes", ww.BytesWritten(),
					"duration", fmt.Sprintf("%.3fms", float64(time.Since(start).Microseconds())/1000),
					"ip", r.RemoteAddr,
					"request_id", chimw.GetReqID(r.Context()),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// csrfMiddleware validates the X-CSRF-Token header against the kantar_csrf cookie
// for state-changing requests (POST, PUT, DELETE, PATCH).
// GET, HEAD, OPTIONS are safe methods and skip validation.
// Requests with a Bearer Authorization header skip validation (CLI/API clients).
func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Safe methods don't need CSRF validation
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		// CLI/API clients using Bearer tokens skip CSRF (they don't send cookies)
		if auth := r.Header.Get("Authorization"); auth != "" {
			next.ServeHTTP(w, r)
			return
		}

		// Browser requests must have matching CSRF token
		cookie, err := r.Cookie("kantar_csrf")
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusForbidden, "missing CSRF token")
			return
		}

		header := r.Header.Get("X-CSRF-Token")
		if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
			writeError(w, http.StatusForbidden, "invalid CSRF token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// generateCSRFToken creates a cryptographically random 32-byte hex token.
func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// setAuthCookies sets the JWT and CSRF cookies on the response.
func setAuthCookies(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	secure := r.TLS != nil
	http.SetCookie(w, &http.Cookie{
		Name:     "kantar_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Expires:  expiresAt,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "kantar_csrf",
		Value:    generateCSRFToken(),
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Expires:  expiresAt,
	})
}

// clearAuthCookies removes the JWT and CSRF cookies.
func clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "kantar_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:   "kantar_csrf",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}
