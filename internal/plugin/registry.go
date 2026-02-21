// Package plugin manages the lifecycle and registration of ecosystem plugins.
package plugin

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

// Registry holds all registered ecosystem plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[registry.EcosystemType]registry.RegistryPlugin
	logger  *slog.Logger
}

// NewRegistry creates a new plugin registry.
func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		plugins: make(map[registry.EcosystemType]registry.RegistryPlugin),
		logger:  logger,
	}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(plugin registry.RegistryPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	eco := plugin.Ecosystem()
	if _, exists := r.plugins[eco]; exists {
		return fmt.Errorf("plugin already registered for ecosystem: %s", eco)
	}

	r.plugins[eco] = plugin
	r.logger.Info("plugin registered",
		"name", plugin.Name(),
		"version", plugin.Version(),
		"ecosystem", eco,
	)

	return nil
}

// Get returns the plugin for the given ecosystem type.
func (r *Registry) Get(ecosystem registry.EcosystemType) (registry.RegistryPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.plugins[ecosystem]
	if !ok {
		return nil, fmt.Errorf("no plugin registered for ecosystem: %s", ecosystem)
	}

	return plugin, nil
}

// List returns all registered plugins.
func (r *Registry) List() []registry.RegistryPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]registry.RegistryPlugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// Has checks if a plugin is registered for the given ecosystem.
func (r *Registry) Has(ecosystem registry.EcosystemType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.plugins[ecosystem]
	return ok
}

// Count returns the number of registered plugins.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.plugins)
}

// ConfigureAll applies configuration to all registered plugins.
func (r *Registry) ConfigureAll(configs map[string]map[string]any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for eco, plugin := range r.plugins {
		cfg, ok := configs[string(eco)]
		if !ok {
			cfg = plugin.DefaultConfig()
		}

		if err := plugin.Configure(cfg); err != nil {
			return fmt.Errorf("configuring plugin %s: %w", eco, err)
		}

		r.logger.Info("plugin configured", "ecosystem", eco)
	}

	return nil
}

// MountRoutes creates a chi router with all plugin routes mounted.
func (r *Registry) MountRoutes(parentRouter chi.Router) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, plugin := range r.plugins {
		routes := plugin.Routes()
		if len(routes) == 0 {
			continue
		}

		subRouter := chi.NewRouter()
		for _, route := range routes {
			switch route.Method {
			case http.MethodGet:
				subRouter.Get(route.Pattern, route.Handler)
			case http.MethodPost:
				subRouter.Post(route.Pattern, route.Handler)
			case http.MethodPut:
				subRouter.Put(route.Pattern, route.Handler)
			case http.MethodDelete:
				subRouter.Delete(route.Pattern, route.Handler)
			case http.MethodHead:
				subRouter.Head(route.Pattern, route.Handler)
			case http.MethodPatch:
				subRouter.Patch(route.Pattern, route.Handler)
			case "":
				// Mount as HandleFunc for all methods
				subRouter.HandleFunc(route.Pattern, route.Handler)
			}
		}

		prefix := "/" + string(plugin.Ecosystem())
		parentRouter.Mount(prefix, subRouter)
		r.logger.Info("plugin routes mounted",
			"ecosystem", plugin.Ecosystem(),
			"prefix", prefix,
			"routes", len(routes),
		)
	}
}
