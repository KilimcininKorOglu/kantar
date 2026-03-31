// Package nuget implements the NuGet V3 registry API plugin for Kantar.
package nuget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/KilimcininKorOglu/kantar/internal/storage"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

// Plugin implements the RegistryPlugin interface for NuGet packages.
type Plugin struct {
	mu      sync.RWMutex
	storage storage.Storage
	logger  *slog.Logger
	config  pluginConfig
}

type pluginConfig struct {
	Upstream string `json:"upstream"`
}

// New creates a new NuGet registry plugin.
func New(store storage.Storage, logger *slog.Logger) *Plugin {
	return &Plugin{
		storage: store,
		logger:  logger,
	}
}

func (p *Plugin) Name() string                      { return "NuGet Registry" }
func (p *Plugin) Ecosystem() registry.EcosystemType { return registry.EcosystemNuGet }

func (p *Plugin) Configure(config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if upstream, ok := config["upstream"].(string); ok {
		p.config.Upstream = upstream
	}
	if p.config.Upstream == "" {
		p.config.Upstream = "https://api.nuget.org"
	}
	return nil
}

func (p *Plugin) DefaultConfig() map[string]any {
	return map[string]any{
		"upstream": "https://api.nuget.org",
	}
}

func (p *Plugin) Search(_ context.Context, query string) ([]registry.PackageMeta, error) {
	return nil, nil
}

func (p *Plugin) FetchPackage(_ context.Context, name, version string) (*registry.Package, error) {
	return nil, fmt.Errorf("not implemented: use upstream sync")
}

func (p *Plugin) FetchMetadata(_ context.Context, name string) (*registry.PackageMeta, error) {
	return &registry.PackageMeta{
		Name:     name,
		Registry: registry.EcosystemNuGet,
	}, nil
}

var nugetHTTPClient = &http.Client{Timeout: 30 * time.Second}

type nugetRegistrationResponse struct {
	CatalogEntry struct {
		ID               string                `json:"id"`
		Version          string                `json:"version"`
		DependencyGroups []nugetDependencyGroup `json:"dependencyGroups"`
	} `json:"catalogEntry"`
}

type nugetDependencyGroup struct {
	TargetFramework string            `json:"targetFramework"`
	Dependencies    []nugetDependency `json:"dependencies"`
}

type nugetDependency struct {
	ID    string `json:"id"`
	Range string `json:"range"`
}

// ResolveDependencies fetches package metadata from the NuGet registration API.
func (p *Plugin) ResolveDependencies(ctx context.Context, name, versionRange string) ([]registry.Dependency, string, error) {
	version := versionRange
	if version == "" || version == "*" || version == "latest" {
		return nil, "", fmt.Errorf("NuGet requires explicit version for %s", name)
	}

	// NuGet registration API
	url := fmt.Sprintf("https://api.nuget.org/v3/registration5-gz-semver2/%s/%s.json",
		strings.ToLower(name), strings.ToLower(version))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := nugetHTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetching NuGet registration for %s@%s: %w", name, version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("NuGet returned %d for %s@%s", resp.StatusCode, name, version)
	}

	var regResp nugetRegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return nil, "", err
	}

	resolvedVersion := regResp.CatalogEntry.Version
	if resolvedVersion == "" {
		resolvedVersion = version
	}

	// Collect deps from all target framework groups
	seen := make(map[string]bool)
	var deps []registry.Dependency
	for _, group := range regResp.CatalogEntry.DependencyGroups {
		for _, d := range group.Dependencies {
			if seen[strings.ToLower(d.ID)] {
				continue
			}
			seen[strings.ToLower(d.ID)] = true
			deps = append(deps, registry.Dependency{
				Name:         d.ID,
				VersionRange: d.Range,
			})
		}
	}

	return deps, resolvedVersion, nil
}

func (p *Plugin) ServePackage(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (p *Plugin) PublishPackage(_ context.Context, _ *registry.Package) error {
	return fmt.Errorf("use NuGet push protocol")
}

func (p *Plugin) DeletePackage(_ context.Context, name, version string) error {
	id := normalizeID(name)
	return p.storage.Delete(
		context.Background(),
		fmt.Sprintf("nuget/packages/%s/%s/%s.%s.nupkg", id, version, id, version),
	)
}

func (p *Plugin) ValidatePackage(_ context.Context, _ *registry.Package) (*registry.ValidationResult, error) {
	return &registry.ValidationResult{Valid: true}, nil
}

func (p *Plugin) Routes() []registry.Route {
	return []registry.Route{
		{Method: http.MethodGet, Pattern: "/v3/index.json", Handler: p.handleServiceIndex},
		{Method: http.MethodGet, Pattern: "/v3/flatcontainer/{id}/index.json", Handler: p.handleVersionList},
		{Method: http.MethodGet, Pattern: "/v3/flatcontainer/{id}/{version}/{filename}", Handler: p.handleDownload},
		{Method: http.MethodPut, Pattern: "/v3/package", Handler: p.handlePush},
		{Method: http.MethodGet, Pattern: "/v3/search", Handler: p.handleSearch},
	}
}

// --- NuGet V3 types ---

// serviceIndex is the NuGet V3 service index response.
type serviceIndex struct {
	Version   string            `json:"version"`
	Resources []serviceResource `json:"resources"`
}

type serviceResource struct {
	ID      string `json:"@id"`
	Type    string `json:"@type"`
	Comment string `json:"comment,omitempty"`
}

// versionList holds the list of versions for a package.
type versionList struct {
	Versions []string `json:"versions"`
}

// searchResponse is the NuGet V3 search endpoint response.
type searchResponse struct {
	TotalHits int         `json:"totalHits"`
	Data      []searchHit `json:"data"`
}

type searchHit struct {
	ID          string   `json:"id"`
	Version     string   `json:"version"`
	Description string   `json:"description,omitempty"`
	Versions    []string `json:"versions,omitempty"`
}

// --- Route Handlers ---

func (p *Plugin) handleServiceIndex(w http.ResponseWriter, r *http.Request) {
	// Build base URL from the incoming request so resource URLs are relative
	// to wherever the plugin is mounted.
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}

	// Determine the mount prefix.  The service index lives at
	// <mount>/v3/index.json, so we strip /v3/index.json from the path.
	mountPrefix := strings.TrimSuffix(r.URL.Path, "/v3/index.json")
	baseURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, mountPrefix)

	idx := serviceIndex{
		Version: "3.0.0",
		Resources: []serviceResource{
			{
				ID:      baseURL + "/v3/flatcontainer/",
				Type:    "PackageBaseAddress/3.0.0",
				Comment: "Base URL of package content",
			},
			{
				ID:      baseURL + "/v3/search",
				Type:    "SearchQueryService/3.5.0",
				Comment: "Search packages",
			},
			{
				ID:      baseURL + "/v3/package",
				Type:    "PackagePublish/2.0.0",
				Comment: "Push packages",
			},
		},
	}
	writeJSON(w, http.StatusOK, idx)
}

func (p *Plugin) handleVersionList(w http.ResponseWriter, r *http.Request) {
	id := normalizeID(extractParam(r, "id"))

	path := fmt.Sprintf("nuget/packages/%s/versions.json", id)
	reader, err := p.storage.Get(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "package not found"})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (p *Plugin) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := normalizeID(extractParam(r, "id"))
	version := extractParam(r, "version")
	filename := extractParam(r, "filename")

	expected := fmt.Sprintf("%s.%s.nupkg", id, version)
	if normalizeID(filename) != expected {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid filename"})
		return
	}

	path := fmt.Sprintf("nuget/packages/%s/%s/%s.%s.nupkg", id, version, id, version)
	reader, err := p.storage.Get(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "package not found"})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", expected))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (p *Plugin) handlePush(w http.ResponseWriter, r *http.Request) {
	// NuGet push uses multipart/form-data with a file field.
	const maxUploadSize = 256 << 20 // 256 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	file, _, err := r.FormFile("package")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing package file"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read package"})
		return
	}

	// Extract package ID and version from the X-NuGet-Package-Id and
	// X-NuGet-Package-Version headers. These are non-standard convenience
	// headers our registry accepts. In a production scenario, the .nuspec
	// inside the .nupkg would be parsed.
	id := normalizeID(r.Header.Get("X-NuGet-Package-Id"))
	version := r.Header.Get("X-NuGet-Package-Version")

	if id == "" || version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "X-NuGet-Package-Id and X-NuGet-Package-Version headers are required",
		})
		return
	}

	// Store the .nupkg file.
	nupkgPath := fmt.Sprintf("nuget/packages/%s/%s/%s.%s.nupkg", id, version, id, version)
	if err := p.storage.Put(r.Context(), nupkgPath, bytes.NewReader(data)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store package"})
		return
	}

	// Update versions.json — read existing, append if not present, write back.
	if err := p.addVersion(r.Context(), id, version); err != nil {
		p.logger.Error("failed to update version list", "id", id, "version", version, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update version list"})
		return
	}

	p.logger.Info("nuget package pushed", "id", id, "version", version)
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id, "version": version})
}

func (p *Plugin) handleSearch(w http.ResponseWriter, r *http.Request) {
	// Minimal search: list all known packages. A full implementation would
	// query an index, but for now we return an empty result set.
	writeJSON(w, http.StatusOK, searchResponse{
		TotalHits: 0,
		Data:      []searchHit{},
	})
}

// addVersion reads the existing versions.json for a package, appends the new
// version if not already present, and writes it back.
func (p *Plugin) addVersion(ctx context.Context, id, version string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	path := fmt.Sprintf("nuget/packages/%s/versions.json", id)

	var vl versionList

	reader, err := p.storage.Get(ctx, path)
	if err == nil {
		defer reader.Close()
		data, readErr := io.ReadAll(reader)
		if readErr == nil {
			json.Unmarshal(data, &vl)
		}
	}

	// Check for duplicates.
	for _, v := range vl.Versions {
		if v == version {
			return nil
		}
	}

	vl.Versions = append(vl.Versions, version)

	encoded, err := json.Marshal(vl)
	if err != nil {
		return fmt.Errorf("marshal versions: %w", err)
	}

	return p.storage.Put(ctx, path, bytes.NewReader(encoded))
}
