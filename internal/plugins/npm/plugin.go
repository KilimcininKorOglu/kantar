// Package npm implements the npm registry API plugin for Kantar.
package npm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/KilimcininKorOglu/kantar/internal/cache"
	"github.com/KilimcininKorOglu/kantar/internal/storage"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

// Plugin implements the RegistryPlugin interface for npm packages.
type Plugin struct {
	mu       sync.RWMutex
	storage  storage.Storage
	logger   *slog.Logger
	appCache cache.Cache
	config   pluginConfig
}

type pluginConfig struct {
	Upstream string `json:"upstream"`
}

// New creates a new npm registry plugin.
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

func (p *Plugin) Name() string                      { return "npm Registry" }
func (p *Plugin) Ecosystem() registry.EcosystemType { return registry.EcosystemNPM }

func (p *Plugin) Configure(config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if upstream, ok := config["upstream"].(string); ok {
		p.config.Upstream = upstream
	}
	if p.config.Upstream == "" {
		p.config.Upstream = "https://registry.npmjs.org"
	}
	return nil
}

func (p *Plugin) DefaultConfig() map[string]any {
	return map[string]any{
		"upstream": "https://registry.npmjs.org",
	}
}

func (p *Plugin) Search(_ context.Context, _ string) ([]registry.PackageMeta, error) {
	return nil, nil
}

func (p *Plugin) FetchPackage(_ context.Context, name, version string) (*registry.Package, error) {
	return nil, fmt.Errorf("not implemented: use upstream sync")
}

func (p *Plugin) FetchMetadata(ctx context.Context, name string) (*registry.PackageMeta, error) {
	packument, err := p.fetchPackument(ctx, name)
	if err != nil {
		return nil, err
	}

	meta := &registry.PackageMeta{
		Name:        packument.Name,
		Description: packument.Description,
		Registry:    registry.EcosystemNPM,
	}

	for ver := range packument.Versions {
		meta.Versions = append(meta.Versions, registry.VersionInfo{
			Version: ver,
		})
	}

	return meta, nil
}

// fetchPackument fetches the full packument from the upstream npm registry.
func (p *Plugin) fetchPackument(ctx context.Context, name string) (*Packument, error) {
	p.mu.RLock()
	upstream := p.config.Upstream
	p.mu.RUnlock()

	if upstream == "" {
		upstream = "https://registry.npmjs.org"
	}

	url := fmt.Sprintf("%s/%s", upstream, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", name, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching packument for %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %d for %s", resp.StatusCode, name)
	}

	var packument Packument
	if err := json.NewDecoder(resp.Body).Decode(&packument); err != nil {
		return nil, fmt.Errorf("decoding packument for %s: %w", name, err)
	}

	return &packument, nil
}

// ResolveDependencies fetches the packument for name, resolves versionRange
// to a concrete version, and returns the runtime dependencies of that version.
func (p *Plugin) ResolveDependencies(ctx context.Context, name, versionRange string) ([]registry.Dependency, string, error) {
	packument, err := p.fetchPackument(ctx, name)
	if err != nil {
		return nil, "", err
	}

	// Collect all version strings
	candidates := make([]string, 0, len(packument.Versions))
	for ver := range packument.Versions {
		candidates = append(candidates, ver)
	}

	// Resolve "latest" via dist-tags
	if versionRange == "latest" || versionRange == "" {
		if latest, ok := packument.DistTags["latest"]; ok {
			versionRange = latest
		}
	}

	// Import selectBestVersion from sync package would create circular dep.
	// Instead, find the best matching version inline.
	resolvedVersion := findBestVersion(candidates, versionRange)
	if resolvedVersion == "" {
		return nil, "", fmt.Errorf("no version of %s matches %s", name, versionRange)
	}

	vDoc, ok := packument.Versions[resolvedVersion]
	if !ok {
		return nil, "", fmt.Errorf("version %s not found in packument for %s", resolvedVersion, name)
	}

	var deps []registry.Dependency
	for depName, depRange := range vDoc.Dependencies {
		deps = append(deps, registry.Dependency{
			Name:         depName,
			VersionRange: depRange,
		})
	}

	return deps, resolvedVersion, nil
}

func (p *Plugin) ServePackage(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (p *Plugin) PublishPackage(_ context.Context, _ *registry.Package) error {
	return fmt.Errorf("use npm publish protocol")
}

func (p *Plugin) DeletePackage(_ context.Context, name, version string) error {
	return p.storage.Delete(context.Background(), fmt.Sprintf("npm/packages/%s/%s", name, version))
}

func (p *Plugin) ValidatePackage(_ context.Context, _ *registry.Package) (*registry.ValidationResult, error) {
	return &registry.ValidationResult{Valid: true}, nil
}

func (p *Plugin) Routes() []registry.Route {
	return []registry.Route{
		{Method: http.MethodGet, Pattern: "/{package}", Handler: p.handleGetPackage},
		{Method: http.MethodGet, Pattern: "/{package}/{version}", Handler: p.handleGetVersion},
		{Method: http.MethodGet, Pattern: "/{package}/-/{tarball}", Handler: p.handleGetTarball},
		{Method: http.MethodPut, Pattern: "/{package}", Handler: p.handlePublish},
		{Method: http.MethodGet, Pattern: "/-/v1/search", Handler: p.handleSearch},
	}
}

// --- Packument (Package Document) ---

// Packument is the npm registry package document format.
type Packument struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	DistTags    map[string]string     `json:"dist-tags,omitempty"`
	Versions    map[string]VersionDoc `json:"versions,omitempty"`
	Time        map[string]string     `json:"time,omitempty"`
	License     string                `json:"license,omitempty"`
}

// VersionDoc is a single version entry in a packument.
type VersionDoc struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description,omitempty"`
	License      string            `json:"license,omitempty"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
	Dist         DistInfo          `json:"dist"`
}

// DistInfo holds distribution information for a package version.
type DistInfo struct {
	Tarball   string `json:"tarball"`
	Shasum    string `json:"shasum,omitempty"`
	Integrity string `json:"integrity,omitempty"`
}

// --- Route Handlers ---

func (p *Plugin) handleGetPackage(w http.ResponseWriter, r *http.Request) {
	name := extractParam(r, "package")

	path := fmt.Sprintf("npm/packages/%s/packument.json", name)
	reader, err := p.storage.Get(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (p *Plugin) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	name := extractParam(r, "package")
	version := extractParam(r, "version")

	path := fmt.Sprintf("npm/packages/%s/packument.json", name)
	reader, err := p.storage.Get(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	defer reader.Close()

	data, _ := io.ReadAll(reader)
	var packument Packument
	if err := json.Unmarshal(data, &packument); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "corrupt metadata"})
		return
	}

	vDoc, ok := packument.Versions[version]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("version %s not found for %s", version, name)})
		return
	}

	writeJSON(w, http.StatusOK, vDoc)
}

func (p *Plugin) handleGetTarball(w http.ResponseWriter, r *http.Request) {
	name := extractParam(r, "package")
	tarball := extractParam(r, "tarball")

	path := fmt.Sprintf("npm/tarballs/%s/%s", name, tarball)
	reader, err := p.storage.Get(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tarball not found"})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (p *Plugin) handlePublish(w http.ResponseWriter, r *http.Request) {
	name := extractParam(r, "package")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}

	var packument Packument
	if err := json.Unmarshal(body, &packument); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	// Store the packument
	path := fmt.Sprintf("npm/packages/%s/packument.json", name)
	if err := p.storage.Put(r.Context(), path, bytesReader(body)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store"})
		return
	}

	// Extract and store attachments (tarballs) if present
	var publishDoc struct {
		Attachments map[string]struct {
			Data string `json:"data"`
		} `json:"_attachments"`
	}
	json.Unmarshal(body, &publishDoc)

	for filename, attachment := range publishDoc.Attachments {
		if attachment.Data != "" {
			tarballPath := fmt.Sprintf("npm/tarballs/%s/%s", name, filename)
			decoded := decodeBase64(attachment.Data)
			if decoded != nil {
				p.storage.Put(r.Context(), tarballPath, bytesReader(decoded))
			}
		}
	}

	p.logger.Info("npm package published", "name", name)
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "success": true})
}

func (p *Plugin) handleSearch(w http.ResponseWriter, r *http.Request) {
	// Minimal search response
	writeJSON(w, http.StatusOK, map[string]any{
		"objects": []any{},
		"total":   0,
	})
}
