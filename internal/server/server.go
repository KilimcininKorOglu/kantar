// Package server provides the HTTP server for Kantar.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/KilimcininKorOglu/kantar/internal/config"
)

// Server is the main HTTP server for Kantar.
type Server struct {
	router chi.Router
	config config.ServerConfig
	srv    *http.Server
	logger *slog.Logger
}

// New creates a new Server instance.
func New(cfg config.ServerConfig, logger *slog.Logger) *Server {
	r := chi.NewRouter()

	s := &Server{
		router: r,
		config: cfg,
		logger: logger,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// Router returns the underlying chi router for mounting plugin routes.
func (s *Server) Router() chi.Router {
	return s.router
}

// Start begins listening and serving HTTP requests.
// It blocks until the context is cancelled or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.srv = &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("server starting", "addr", addr)

		var err error
		if s.config.TLSCert != "" && s.config.TLSKey != "" {
			err = s.srv.ListenAndServeTLS(s.config.TLSCert, s.config.TLSKey)
		} else {
			err = s.srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("server shutting down")

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return s.srv.Shutdown(shutdownCtx)
}

func (s *Server) setupMiddleware() {
	s.router.Use(chimw.RequestID)
	s.router.Use(chimw.RealIP)
	s.router.Use(newStructuredLogger(s.logger))
	s.router.Use(chimw.Recoverer)
	s.router.Use(chimw.Timeout(60 * time.Second))
	s.router.Use(chimw.Compress(5))
}

func (s *Server) setupRoutes() {
	s.router.Get("/healthz", s.handleHealthz)

	// Management API
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/system/status", s.handleSystemStatus)
	})
}

// MountPluginRoutes mounts routes from a plugin under the given prefix.
func (s *Server) MountPluginRoutes(prefix string, handler http.Handler) {
	s.router.Mount(prefix, handler)
	s.logger.Info("plugin routes mounted", "prefix", prefix)
}
