// Package helm implements the Helm chart registry plugin for Kantar.
package helm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"archive/tar"
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/KilimcininKorOglu/kantar/internal/storage"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

// Plugin implements the RegistryPlugin interface for Helm chart repositories.
type Plugin struct {
	mu      sync.RWMutex
	storage storage.Storage
	logger  *slog.Logger
	config  pluginConfig
}

type pluginConfig struct {
	Upstream string `json:"upstream"`
}

// chartMeta holds metadata for a single chart version, stored as JSON.
type chartMeta struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description,omitempty"`
	AppVersion  string    `json:"appVersion,omitempty"`
	Digest      string    `json:"digest"`
	Created     time.Time `json:"created"`
	URLs        []string  `json:"urls"`
}

// chartIndex holds all versions of a single chart name, stored as JSON.
type chartIndex struct {
	Name     string      `json:"name"`
	Versions []chartMeta `json:"versions"`
}

// New creates a new Helm chart registry plugin.
func New(store storage.Storage, logger *slog.Logger) *Plugin {
	return &Plugin{
		storage: store,
		logger:  logger,
	}
}

func (p *Plugin) Name() string                      { return "Helm Chart Registry" }
func (p *Plugin) Ecosystem() registry.EcosystemType { return registry.EcosystemHelm }

func (p *Plugin) Configure(config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if upstream, ok := config["upstream"].(string); ok {
		p.config.Upstream = upstream
	}
	return nil
}

func (p *Plugin) DefaultConfig() map[string]any {
	return map[string]any{
		"upstream": "",
	}
}

func (p *Plugin) Search(_ context.Context, _ string) ([]registry.PackageMeta, error) {
	return nil, nil
}

func (p *Plugin) FetchPackage(_ context.Context, _, _ string) (*registry.Package, error) {
	return nil, fmt.Errorf("helm charts are pushed, not fetched from upstream")
}

func (p *Plugin) FetchMetadata(_ context.Context, name string) (*registry.PackageMeta, error) {
	return &registry.PackageMeta{
		Name:     name,
		Registry: registry.EcosystemHelm,
	}, nil
}

// ResolveDependencies opens the stored chart .tgz and parses dependencies from Chart.yaml.
func (p *Plugin) ResolveDependencies(ctx context.Context, name, versionRange string) ([]registry.Dependency, string, error) {
	version := versionRange
	if version == "" || version == "*" || version == "latest" {
		// Try to find latest version from stored metadata
		return nil, "", fmt.Errorf("Helm requires explicit version for %s", name)
	}

	// Read chart .tgz from storage
	chartPath := fmt.Sprintf("helm/charts/%s/%s-%s.tgz", name, name, version)
	reader, err := p.storage.Get(ctx, chartPath)
	if err != nil {
		return nil, "", fmt.Errorf("chart not found: %s@%s", name, version)
	}
	defer reader.Close()

	chartYAML, err := extractChartYAML(reader, name)
	if err != nil {
		return nil, "", fmt.Errorf("reading Chart.yaml from %s@%s: %w", name, version, err)
	}

	deps := parseChartDependencies(chartYAML)
	return deps, version, nil
}

// extractChartYAML opens a .tgz and finds {chartName}/Chart.yaml
func extractChartYAML(r io.Reader, chartName string) (string, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		// Match {chartName}/Chart.yaml or Chart.yaml
		if hdr.Name == chartName+"/Chart.yaml" || hdr.Name == "Chart.yaml" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}
	return "", fmt.Errorf("Chart.yaml not found in archive")
}

// parseChartDependencies extracts dependencies from Chart.yaml content
// using simple line-based parsing (avoids YAML library dependency).
func parseChartDependencies(content string) []registry.Dependency {
	var deps []registry.Dependency
	lines := strings.Split(content, "\n")
	inDeps := false
	var currentDep registry.Dependency
	hasDep := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect dependencies: block
		if trimmed == "dependencies:" {
			inDeps = true
			continue
		}

		// Exit dependencies block on next top-level key
		if inDeps && len(line) > 0 && line[0] != ' ' && line[0] != '-' {
			if hasDep {
				deps = append(deps, currentDep)
				hasDep = false
			}
			break
		}

		if !inDeps {
			continue
		}

		// New dependency entry
		if strings.HasPrefix(trimmed, "- name:") {
			if hasDep {
				deps = append(deps, currentDep)
			}
			currentDep = registry.Dependency{
				Name: strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")),
			}
			hasDep = true
			continue
		}

		if hasDep {
			if strings.HasPrefix(trimmed, "version:") {
				currentDep.VersionRange = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "version:")), "\"'")
			}
		}
	}

	if hasDep {
		deps = append(deps, currentDep)
	}

	return deps
}

func (p *Plugin) ServePackage(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (p *Plugin) PublishPackage(_ context.Context, _ *registry.Package) error {
	return fmt.Errorf("use Helm chart upload API")
}

func (p *Plugin) DeletePackage(ctx context.Context, name, version string) error {
	return p.deleteChart(ctx, name, version)
}

func (p *Plugin) ValidatePackage(_ context.Context, _ *registry.Package) (*registry.ValidationResult, error) {
	return &registry.ValidationResult{Valid: true}, nil
}

func (p *Plugin) Routes() []registry.Route {
	return []registry.Route{
		{Method: http.MethodGet, Pattern: "/index.yaml", Handler: p.handleGetIndex},
		{Method: http.MethodGet, Pattern: "/charts/{filename}", Handler: p.handleDownloadChart},
		{Method: http.MethodPost, Pattern: "/api/charts", Handler: p.handleUploadChart},
		{Method: http.MethodDelete, Pattern: "/api/charts/{name}/{version}", Handler: p.handleDeleteChart},
	}
}

// --- Route Handlers ---

func (p *Plugin) handleGetIndex(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	entries, err := p.loadAllMetadata(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load chart metadata"})
		return
	}

	yaml := generateIndexYAML(entries)
	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(yaml))
}

func (p *Plugin) handleDownloadChart(w http.ResponseWriter, r *http.Request) {
	filename := extractParam(r, "filename")

	chartName, version := parseChartFilename(filename)
	if chartName == "" || version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid chart filename"})
		return
	}

	path := fmt.Sprintf("helm/charts/%s/%s-%s.tgz", chartName, chartName, version)
	reader, err := p.storage.Get(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "chart not found"})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/gzip")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (p *Plugin) handleUploadChart(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	file, _, err := r.FormFile("chart")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "chart file is required"})
		return
	}
	defer file.Close()

	chartData, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read chart data"})
		return
	}

	// Extract chart name and version from the form or fallback to form fields.
	chartName := r.FormValue("name")
	chartVersion := r.FormValue("version")

	if chartName == "" || chartVersion == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and version form fields are required"})
		return
	}

	// Compute SHA-256 digest.
	hash := sha256.Sum256(chartData)
	digest := hex.EncodeToString(hash[:])

	// Store the chart tarball.
	chartPath := fmt.Sprintf("helm/charts/%s/%s-%s.tgz", chartName, chartName, chartVersion)
	if err := p.storage.Put(r.Context(), chartPath, bytes.NewReader(chartData)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store chart"})
		return
	}

	// Update metadata.
	meta := chartMeta{
		Name:        chartName,
		Version:     chartVersion,
		Description: r.FormValue("description"),
		AppVersion:  r.FormValue("appVersion"),
		Digest:      digest,
		Created:     time.Now().UTC(),
		URLs:        []string{fmt.Sprintf("charts/%s-%s.tgz", chartName, chartVersion)},
	}

	if err := p.addChartVersion(r.Context(), chartName, meta); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update metadata"})
		return
	}

	p.logger.Info("helm chart uploaded", "name", chartName, "version", chartVersion)
	writeJSON(w, http.StatusCreated, map[string]any{"saved": true})
}

func (p *Plugin) handleDeleteChart(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()

	name := extractParam(r, "name")
	version := extractParam(r, "version")

	if name == "" || version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and version are required"})
		return
	}

	if err := p.deleteChart(r.Context(), name, version); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	p.logger.Info("helm chart deleted", "name", name, "version", version)
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// --- Internal Metadata Operations ---

func (p *Plugin) metadataPath(name string) string {
	return fmt.Sprintf("helm/metadata/%s.json", name)
}

func (p *Plugin) loadChartIndex(ctx context.Context, name string) (*chartIndex, error) {
	reader, err := p.storage.Get(ctx, p.metadataPath(name))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading metadata for %s: %w", name, err)
	}

	var idx chartIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing metadata for %s: %w", name, err)
	}
	return &idx, nil
}

func (p *Plugin) saveChartIndex(ctx context.Context, idx *chartIndex) error {
	data, err := json.Marshal(idx)
	if err != nil {
		return fmt.Errorf("serializing metadata for %s: %w", idx.Name, err)
	}
	return p.storage.Put(ctx, p.metadataPath(idx.Name), bytes.NewReader(data))
}

func (p *Plugin) addChartVersion(ctx context.Context, name string, meta chartMeta) error {
	idx, err := p.loadChartIndex(ctx, name)
	if err != nil {
		// New chart, create index.
		idx = &chartIndex{
			Name:     name,
			Versions: []chartMeta{},
		}
	}

	// Replace existing version or append new one.
	replaced := false
	for i, v := range idx.Versions {
		if v.Version == meta.Version {
			idx.Versions[i] = meta
			replaced = true
			break
		}
	}
	if !replaced {
		idx.Versions = append(idx.Versions, meta)
	}

	return p.saveChartIndex(ctx, idx)
}

func (p *Plugin) deleteChart(ctx context.Context, name, version string) error {
	idx, err := p.loadChartIndex(ctx, name)
	if err != nil {
		return fmt.Errorf("chart %s not found", name)
	}

	found := false
	filtered := make([]chartMeta, 0, len(idx.Versions))
	for _, v := range idx.Versions {
		if v.Version == version {
			found = true
			continue
		}
		filtered = append(filtered, v)
	}

	if !found {
		return fmt.Errorf("version %s not found for chart %s", version, name)
	}

	// Delete the tarball.
	chartPath := fmt.Sprintf("helm/charts/%s/%s-%s.tgz", name, name, version)
	p.storage.Delete(ctx, chartPath)

	if len(filtered) == 0 {
		// No versions left, remove metadata file entirely.
		return p.storage.Delete(ctx, p.metadataPath(name))
	}

	idx.Versions = filtered
	return p.saveChartIndex(ctx, idx)
}

func (p *Plugin) loadAllMetadata(ctx context.Context) (map[string][]chartMeta, error) {
	files, err := p.storage.List(ctx, "helm/metadata")
	if err != nil {
		// No metadata directory yet means no charts.
		return map[string][]chartMeta{}, nil
	}

	entries := make(map[string][]chartMeta)
	for _, f := range files {
		if f.IsDir {
			continue
		}
		reader, err := p.storage.Get(ctx, f.Path)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			continue
		}
		var idx chartIndex
		if err := json.Unmarshal(data, &idx); err != nil {
			continue
		}
		if len(idx.Versions) > 0 {
			entries[idx.Name] = idx.Versions
		}
	}
	return entries, nil
}
