// Package gomod implements the Go Module Proxy plugin for Kantar.
// It follows the GOPROXY protocol specification to serve Go module
// version lists, version info, go.mod files, and module zip archives.
package gomod

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KilimcininKorOglu/kantar/internal/cache"
	"github.com/KilimcininKorOglu/kantar/internal/httpclient"
	"github.com/KilimcininKorOglu/kantar/internal/storage"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

// Plugin implements the RegistryPlugin interface for Go modules.
type Plugin struct {
	mu      sync.RWMutex
	storage storage.Storage
	logger   *slog.Logger
	appCache cache.Cache
	config   pluginConfig
}

type pluginConfig struct {
	Upstream string `json:"upstream"`
}

// VersionInfo is the JSON structure returned by the GOPROXY .info endpoint.
type VersionInfo struct {
	Version string    `json:"Version"`
	Time    time.Time `json:"Time"`
}

// New creates a new Go module proxy plugin.
func New(store storage.Storage, logger *slog.Logger) *Plugin {
	return &Plugin{
		storage: store,
		logger:  logger,
	}
}

// WithCache sets the cache for upstream response caching.
func (p *Plugin) WithCache(c cache.Cache) {
	p.appCache = c
}

func (p *Plugin) Name() string                      { return "Go Module Proxy" }
func (p *Plugin) Ecosystem() registry.EcosystemType { return registry.EcosystemGoMod }

func (p *Plugin) Configure(config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if upstream, ok := config["upstream"].(string); ok {
		p.config.Upstream = upstream
	}
	if p.config.Upstream == "" {
		p.config.Upstream = "https://proxy.golang.org"
	}
	return nil
}

func (p *Plugin) DefaultConfig() map[string]any {
	return map[string]any{
		"upstream": "https://proxy.golang.org",
	}
}

func (p *Plugin) Search(_ context.Context, _ string) ([]registry.PackageMeta, error) {
	return nil, nil
}

func (p *Plugin) FetchPackage(_ context.Context, _, _ string) (*registry.Package, error) {
	return nil, fmt.Errorf("not implemented: use GOPROXY protocol")
}

func (p *Plugin) FetchMetadata(_ context.Context, name string) (*registry.PackageMeta, error) {
	return &registry.PackageMeta{
		Name:     name,
		Registry: registry.EcosystemGoMod,
	}, nil
}

var gomodHTTPClient = httpclient.Shared

// ResolveDependencies fetches the go.mod file from upstream and parses require directives.
func (p *Plugin) ResolveDependencies(ctx context.Context, name, versionRange string) ([]registry.Dependency, string, error) {
	p.mu.RLock()
	upstream := p.config.Upstream
	p.mu.RUnlock()
	if upstream == "" {
		upstream = "https://proxy.golang.org"
	}

	version := versionRange
	if version == "" || version == "*" || version == "latest" {
		// Fetch latest version from list
		url := fmt.Sprintf("%s/%s/@v/list", upstream, name)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, "", err
		}
		resp, err := gomodHTTPClient.Do(req)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) == 0 || lines[0] == "" {
			return nil, "", fmt.Errorf("no versions found for %s", name)
		}
		sort.Slice(lines, func(i, j int) bool {
			return compareGoVersion(lines[i], lines[j]) < 0
		})
		version = lines[len(lines)-1]
	}

	// Cache the go.mod body for the resolved version
	cacheKey := fmt.Sprintf("upstream:%s:%s@%s", p.Ecosystem(), name, version)

	var modData []byte

	// Try cache first
	if p.appCache != nil {
		if cached, _ := p.appCache.Get(ctx, cacheKey); cached != nil {
			modData = cached
		}
	}

	if modData == nil {
		// Fetch go.mod from upstream
		url := fmt.Sprintf("%s/%s/@v/%s.mod", upstream, name, version)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, "", err
		}

		resp, err := gomodHTTPClient.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("fetching go.mod for %s@%s: %w", name, version, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("upstream returned %d for %s@%s.mod", resp.StatusCode, name, version)
		}

		var readErr error
		modData, readErr = io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, "", readErr
		}

		if p.appCache != nil {
			p.appCache.Set(ctx, cacheKey, modData, 5*time.Minute)
		}
	}

	deps := parseGoMod(string(modData))
	return deps, version, nil
}

// parseGoMod extracts require directives from a go.mod file content.
func parseGoMod(content string) []registry.Dependency {
	var deps []registry.Dependency
	inRequireBlock := false

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)

		if line == "require (" {
			inRequireBlock = true
			continue
		}
		if line == ")" && inRequireBlock {
			inRequireBlock = false
			continue
		}

		// Single-line require
		if strings.HasPrefix(line, "require ") && !inRequireBlock {
			parts := strings.Fields(line[8:])
			if len(parts) >= 2 {
				deps = append(deps, registry.Dependency{
					Name:         parts[0],
					VersionRange: parts[1],
				})
			}
			continue
		}

		if inRequireBlock {
			// Skip comments and indirect markers
			if strings.HasPrefix(line, "//") || line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				dep := registry.Dependency{
					Name:         parts[0],
					VersionRange: parts[1],
				}
				// Mark indirect deps
				if strings.Contains(line, "// indirect") {
					dep.Optional = true
				}
				deps = append(deps, dep)
			}
		}
	}

	return deps
}

func (p *Plugin) ServePackage(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (p *Plugin) PublishPackage(_ context.Context, _ *registry.Package) error {
	return fmt.Errorf("use GOPROXY protocol")
}

func (p *Plugin) DeletePackage(_ context.Context, name, version string) error {
	encoded := encodePath(name)
	prefix := fmt.Sprintf("gomod/modules/%s/%s", encoded, version)

	for _, suffix := range []string{".info", ".mod", ".zip"} {
		_ = p.storage.Delete(context.Background(), prefix+suffix)
	}
	return nil
}

func (p *Plugin) ValidatePackage(_ context.Context, _ *registry.Package) (*registry.ValidationResult, error) {
	return &registry.ValidationResult{Valid: true}, nil
}

func (p *Plugin) Routes() []registry.Route {
	return []registry.Route{
		{Method: http.MethodGet, Pattern: "/*", Handler: p.handleProxy},
		{Method: http.MethodPut, Pattern: "/*", Handler: p.handleUpload},
	}
}

// --- Route Handlers ---

func (p *Plugin) handleProxy(w http.ResponseWriter, r *http.Request) {
	// The chi wildcard captures everything after the mount prefix.
	raw := extractParam(r, "*")
	raw = strings.TrimPrefix(raw, "/")

	// GET /{module}/@latest
	if strings.HasSuffix(raw, "/@latest") {
		modulePath := strings.TrimSuffix(raw, "/@latest")
		p.handleLatest(w, r, modulePath)
		return
	}

	// Paths under /@v/
	idx := strings.LastIndex(raw, "/@v/")
	if idx < 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "invalid GOPROXY path"})
		return
	}

	modulePath := raw[:idx]
	remainder := raw[idx+len("/@v/"):]

	switch {
	case remainder == "list":
		p.handleList(w, r, modulePath)
	case strings.HasSuffix(remainder, ".info"):
		version := strings.TrimSuffix(remainder, ".info")
		p.handleInfo(w, r, modulePath, version)
	case strings.HasSuffix(remainder, ".mod"):
		version := strings.TrimSuffix(remainder, ".mod")
		p.handleMod(w, r, modulePath, version)
	case strings.HasSuffix(remainder, ".zip"):
		version := strings.TrimSuffix(remainder, ".zip")
		p.handleZip(w, r, modulePath, version)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown endpoint"})
	}
}

func (p *Plugin) handleList(w http.ResponseWriter, r *http.Request, modulePath string) {
	encoded := encodePath(modulePath)
	path := fmt.Sprintf("gomod/modules/%s/versions.txt", encoded)

	reader, err := p.storage.Get(r.Context(), path)
	if err != nil {
		// Return empty list rather than 404, per GOPROXY convention.
		writeText(w, http.StatusOK, "")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (p *Plugin) handleInfo(w http.ResponseWriter, r *http.Request, modulePath, version string) {
	encoded := encodePath(modulePath)
	path := fmt.Sprintf("gomod/modules/%s/%s/.info", encoded, version)

	reader, err := p.storage.Get(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "version not found"})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (p *Plugin) handleMod(w http.ResponseWriter, r *http.Request, modulePath, version string) {
	encoded := encodePath(modulePath)
	path := fmt.Sprintf("gomod/modules/%s/%s/.mod", encoded, version)

	reader, err := p.storage.Get(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "go.mod not found"})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (p *Plugin) handleZip(w http.ResponseWriter, r *http.Request, modulePath, version string) {
	encoded := encodePath(modulePath)
	path := fmt.Sprintf("gomod/modules/%s/%s/.zip", encoded, version)

	reader, err := p.storage.Get(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "zip not found"})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (p *Plugin) handleLatest(w http.ResponseWriter, r *http.Request, modulePath string) {
	encoded := encodePath(modulePath)
	versionsPath := fmt.Sprintf("gomod/modules/%s/versions.txt", encoded)

	reader, err := p.storage.Get(r.Context(), versionsPath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "module not found"})
		return
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read versions"})
		return
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no versions available"})
		return
	}

	sort.Slice(lines, func(i, j int) bool {
		return compareGoVersion(lines[i], lines[j]) < 0
	})
	latest := lines[len(lines)-1]

	infoPath := fmt.Sprintf("gomod/modules/%s/%s/.info", encoded, latest)
	infoReader, err := p.storage.Get(r.Context(), infoPath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "latest version info not found"})
		return
	}
	defer infoReader.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, infoReader)
}

// handleUpload handles PUT requests for storing module artifacts.
// PUT /{module}/@v/{version}.{info|mod|zip}
func (p *Plugin) handleUpload(w http.ResponseWriter, r *http.Request) {
	raw := extractParam(r, "*")
	raw = strings.TrimPrefix(raw, "/")

	idx := strings.LastIndex(raw, "/@v/")
	if idx < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid upload path"})
		return
	}

	modulePath := raw[:idx]
	remainder := raw[idx+len("/@v/"):]
	encoded := encodePath(modulePath)

	var suffix string
	var version string

	switch {
	case strings.HasSuffix(remainder, ".info"):
		suffix = ".info"
		version = strings.TrimSuffix(remainder, ".info")
	case strings.HasSuffix(remainder, ".mod"):
		suffix = ".mod"
		version = strings.TrimSuffix(remainder, ".mod")
	case strings.HasSuffix(remainder, ".zip"):
		suffix = ".zip"
		version = strings.TrimSuffix(remainder, ".zip")
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported file type"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}

	storagePath := fmt.Sprintf("gomod/modules/%s/%s/%s", encoded, version, suffix)
	if err := p.storage.Put(r.Context(), storagePath, bytesReader(body)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store"})
		return
	}

	// When uploading .info, update the version list.
	if suffix == ".info" {
		if err := p.appendVersion(r.Context(), encoded, version); err != nil {
			p.logger.Warn("failed to update version list", "module", modulePath, "error", err)
		}
	}

	p.logger.Info("gomod artifact stored", "module", modulePath, "version", version, "type", suffix)
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}

// appendVersion adds a version to the versions.txt file if not already present.
func (p *Plugin) appendVersion(ctx context.Context, encodedModule, version string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	path := fmt.Sprintf("gomod/modules/%s/versions.txt", encodedModule)

	var existing string
	reader, err := p.storage.Get(ctx, path)
	if err == nil {
		data, _ := io.ReadAll(reader)
		reader.Close()
		existing = string(data)
	}

	// Check if version is already listed.
	for _, line := range strings.Split(existing, "\n") {
		if strings.TrimSpace(line) == version {
			return nil
		}
	}

	if existing != "" && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	existing += version + "\n"

	return p.storage.Put(ctx, path, bytesReader([]byte(existing)))
}

// compareGoVersion compares two Go module versions (vMAJOR.MINOR.PATCH format).
// Returns -1, 0, or 1 like strings.Compare.
func compareGoVersion(a, b string) int {
	parseVer := func(v string) (major, minor, patch int) {
		v = strings.TrimPrefix(v, "v")
		if idx := strings.IndexByte(v, '-'); idx >= 0 {
			v = v[:idx] // strip pre-release
		}
		parts := strings.SplitN(v, ".", 3)
		if len(parts) >= 1 {
			major, _ = strconv.Atoi(parts[0])
		}
		if len(parts) >= 2 {
			minor, _ = strconv.Atoi(parts[1])
		}
		if len(parts) >= 3 {
			patch, _ = strconv.Atoi(parts[2])
		}
		return
	}

	aMaj, aMin, aPat := parseVer(a)
	bMaj, bMin, bPat := parseVer(b)

	if aMaj != bMaj {
		if aMaj < bMaj {
			return -1
		}
		return 1
	}
	if aMin != bMin {
		if aMin < bMin {
			return -1
		}
		return 1
	}
	if aPat != bPat {
		if aPat < bPat {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b) // tie-break on full string (pre-release etc.)
}
