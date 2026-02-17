package server

import (
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
