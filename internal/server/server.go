// Package server provides the HTTP server for Kantar.
package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/KilimcininKorOglu/kantar/internal/audit"
	"github.com/KilimcininKorOglu/kantar/internal/auth"
	"github.com/KilimcininKorOglu/kantar/internal/config"
	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
	"github.com/KilimcininKorOglu/kantar/internal/manager"
	syncp "github.com/KilimcininKorOglu/kantar/internal/sync"
)

// Dependencies holds all subsystem dependencies the server needs for API handlers.
type Dependencies struct {
	Queries     *sqlc.Queries
	JWTManager  *auth.JWTManager
	Manager     *manager.Manager
	AuditLogger *audit.Logger
	SyncEngine  *syncp.Engine
}

// Server is the main HTTP server for Kantar.
type Server struct {
	router chi.Router
	config config.ServerConfig
	srv    *http.Server
	logger *slog.Logger
	deps   Dependencies
}

// New creates a new Server instance.
func New(cfg config.ServerConfig, logger *slog.Logger, deps Dependencies) *Server {
	r := chi.NewRouter()

	s := &Server{
		router: r,
		config: cfg,
		logger: logger,
		deps:   deps,
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
	s.router.Route("/api/v1", s.setupAPIRoutes)
}

func (s *Server) setupAPIRoutes(r chi.Router) {
	// Public auth endpoints — no authentication required
	r.Post("/auth/login", s.handleLogin)
	r.Post("/auth/register", s.handleRegister)

	// Authenticated endpoints
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if s.deps.JWTManager == nil {
					writeError(w, http.StatusServiceUnavailable, "authentication service not ready")
					return
				}
				auth.Middleware(s.deps.JWTManager)(next).ServeHTTP(w, r)
			})
		})

		r.Get("/system/status", s.handleSystemStatus)

		// Profile — any authenticated user can manage their own profile
		r.Route("/profile", func(r chi.Router) {
			r.Get("/", s.handleGetProfile)
			r.Put("/", s.handleUpdateProfile)
			r.Put("/password", s.handleChangePassword)
		})

		// User management — super_admin only
		r.Route("/users", func(r chi.Router) {
			r.Use(auth.RequireRole(auth.RoleSuperAdmin))
			r.Get("/", s.handleListUsers)
			r.Get("/{id}", s.handleGetUser)
			r.Put("/{id}", s.handleUpdateUser)
			r.Delete("/{id}", s.handleDeleteUser)
		})

		// Package management
		r.Route("/packages", func(r chi.Router) {
			r.Use(auth.RequireRole(auth.RoleConsumer))
			r.Get("/", s.handleListPackages)
			r.Get("/by-name/{registry}/{name}", s.handleGetPackageByName)
			r.Get("/{id}", s.handleGetPackage)

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(auth.RoleRegistryAdmin))
				r.Post("/{id}/approve", s.handleApprovePackage)
				r.Post("/{id}/block", s.handleBlockPackage)
			})
		})

		// Audit logs — registry_admin+
		r.Route("/audit", func(r chi.Router) {
			r.Use(auth.RequireRole(auth.RoleRegistryAdmin))
			r.Get("/", s.handleListAuditLogs)
			r.Get("/verify", s.handleVerifyAuditChain)
		})

		// Settings — registry_admin+ to read, super_admin to write
		r.Route("/settings", func(r chi.Router) {
			r.Use(auth.RequireRole(auth.RoleRegistryAdmin))
			r.Get("/", s.handleListSettings)
			r.Get("/{key}", s.handleGetSetting)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(auth.RoleSuperAdmin))
				r.Put("/", s.handleBulkUpdateSettings)
				r.Put("/{key}", s.handleUpdateSetting)
			})
		})

		// Registries — consumer+ to read, super_admin to write
		r.Route("/registries", func(r chi.Router) {
			r.Use(auth.RequireRole(auth.RoleConsumer))
			r.Get("/", s.handleListRegistries)
			r.Get("/{ecosystem}", s.handleGetRegistry)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(auth.RoleSuperAdmin))
				r.Put("/{ecosystem}", s.handleUpdateRegistry)
			})
		})

		// Policies — consumer+ to read, super_admin to write
		r.Route("/policies", func(r chi.Router) {
			r.Use(auth.RequireRole(auth.RoleConsumer))
			r.Get("/", s.handleListPolicies)
			r.Get("/{name}", s.handleGetPolicy)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(auth.RoleSuperAdmin))
				r.Put("/{name}", s.handleUpdatePolicy)
				r.Put("/{name}/toggle", s.handleTogglePolicy)
			})
		})

		// Sync jobs — registry_admin+
		r.Route("/sync/jobs", func(r chi.Router) {
			r.Use(auth.RequireRole(auth.RoleRegistryAdmin))
			r.Get("/", s.handleListSyncJobs)
			r.Get("/{id}", s.handleGetSyncJob)
		})
	})
}

// MountPluginRoutes mounts routes from a plugin under the given prefix.
func (s *Server) MountPluginRoutes(prefix string, handler http.Handler) {
	s.router.Mount(prefix, handler)
	s.logger.Info("plugin routes mounted", "prefix", prefix)
}

// MountWebUI serves the embedded SPA from the given filesystem.
// Unknown paths fall back to index.html for client-side routing.
func (s *Server) MountWebUI(webFS fs.FS) {
	fileServer := http.FileServer(http.FS(webFS))

	s.router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		f, err := webFS.Open(r.URL.Path[1:]) // strip leading /
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
