// Package cargo implements the Cargo registry (Sparse Index, RFC 2789) plugin for Kantar.
package cargo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/KilimcininKorOglu/kantar/internal/cache"
	"github.com/KilimcininKorOglu/kantar/internal/storage"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

// Plugin implements the RegistryPlugin interface for Cargo crates.
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

// indexEntry represents a single version entry in the Cargo sparse index.
// Each version is stored as one JSON line in the index file.
type indexEntry struct {
	Name     string              `json:"name"`
	Vers     string              `json:"vers"`
	Deps     []indexDep          `json:"deps"`
	Cksum    string              `json:"cksum"`
	Features map[string][]string `json:"features"`
	Yanked   bool                `json:"yanked"`
	Links    string              `json:"links,omitempty"`
}

// indexDep represents a dependency in a Cargo index entry.
type indexDep struct {
	Name            string   `json:"name"`
	Req             string   `json:"req"`
	Features        []string `json:"features"`
	Optional        bool     `json:"optional"`
	DefaultFeatures bool     `json:"default_features"`
	Target          string   `json:"target,omitempty"`
	Kind            string   `json:"kind,omitempty"`
	Registry        string   `json:"registry,omitempty"`
	Package         string   `json:"package,omitempty"`
}

// publishRequest is the JSON metadata sent during crate publish.
type publishRequest struct {
	Name     string              `json:"name"`
	Vers     string              `json:"vers"`
	Deps     []publishDep        `json:"deps"`
	Features map[string][]string `json:"features"`
	Links    string              `json:"links,omitempty"`
	License  string              `json:"license,omitempty"`
	Desc     string              `json:"description,omitempty"`
}

// publishDep is a dependency in a publish request.
type publishDep struct {
	Name               string   `json:"name"`
	VersionReq         string   `json:"version_req"`
	Features           []string `json:"features"`
	Optional           bool     `json:"optional"`
	DefaultFeatures    bool     `json:"default_features"`
	Target             string   `json:"target,omitempty"`
	Kind               string   `json:"kind,omitempty"`
	Registry           string   `json:"registry,omitempty"`
	ExplicitNameInToml string   `json:"explicit_name_in_toml,omitempty"`
}

// New creates a new Cargo registry plugin.
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

func (p *Plugin) Name() string                      { return "Cargo Registry" }
func (p *Plugin) Ecosystem() registry.EcosystemType { return registry.EcosystemCargo }

func (p *Plugin) Configure(config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if upstream, ok := config["upstream"].(string); ok {
		p.config.Upstream = upstream
	}
	if p.config.Upstream == "" {
		p.config.Upstream = "https://crates.io"
	}
	return nil
}

func (p *Plugin) DefaultConfig() map[string]any {
	return map[string]any{
		"upstream": "https://crates.io",
	}
}

func (p *Plugin) Search(_ context.Context, _ string) ([]registry.PackageMeta, error) {
	return nil, nil
}

func (p *Plugin) FetchPackage(_ context.Context, _, _ string) (*registry.Package, error) {
	return nil, fmt.Errorf("use Cargo download endpoint")
}

func (p *Plugin) FetchMetadata(_ context.Context, name string) (*registry.PackageMeta, error) {
	return &registry.PackageMeta{
		Name:     name,
		Registry: registry.EcosystemCargo,
	}, nil
}

var cargoHTTPClient = &http.Client{Timeout: 30 * time.Second}

// ResolveDependencies fetches the crate index from upstream and returns dependencies
// for the best matching version.
func (p *Plugin) ResolveDependencies(ctx context.Context, name, versionRange string) ([]registry.Dependency, string, error) {
	p.mu.RLock()
	upstream := p.config.Upstream
	p.mu.RUnlock()

	// Derive sparse index URL from configured upstream
	indexBase := "https://index.crates.io"
	if upstream != "" && upstream != "https://crates.io" {
		indexBase = strings.TrimSuffix(upstream, "/")
	}

	prefix := computePrefix(strings.ToLower(name))
	url := fmt.Sprintf("%s/%s%s", indexBase, prefix, strings.ToLower(name))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := cargoHTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetching crate index for %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("upstream returned %d for crate %s", resp.StatusCode, name)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Parse JSON lines
	var entries []indexEntry
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var entry indexEntry
		if json.Unmarshal(line, &entry) == nil && !entry.Yanked {
			entries = append(entries, entry)
		}
	}

	if len(entries) == 0 {
		return nil, "", fmt.Errorf("no versions found for crate %s", name)
	}

	// Pick best version — use last entry (highest version in Cargo index order)
	// For exact version, find it directly
	selected := entries[len(entries)-1]
	if versionRange != "" && versionRange != "*" && versionRange != "latest" {
		for _, e := range entries {
			if e.Vers == versionRange {
				selected = e
				break
			}
		}
	}

	var deps []registry.Dependency
	for _, d := range selected.Deps {
		kind := d.Kind
		if kind == "" {
			kind = "normal"
		}
		deps = append(deps, registry.Dependency{
			Name:         d.Name,
			VersionRange: d.Req,
			Optional:     d.Optional,
			Dev:          kind == "dev",
		})
	}

	return deps, selected.Vers, nil
}

func (p *Plugin) ServePackage(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (p *Plugin) PublishPackage(_ context.Context, _ *registry.Package) error {
	return fmt.Errorf("use Cargo publish protocol")
}

func (p *Plugin) DeletePackage(_ context.Context, name, version string) error {
	return p.storage.Delete(context.Background(), fmt.Sprintf("cargo/crates/%s/%s.crate", name, version))
}

func (p *Plugin) ValidatePackage(_ context.Context, _ *registry.Package) (*registry.ValidationResult, error) {
	return &registry.ValidationResult{Valid: true}, nil
}

func (p *Plugin) Routes() []registry.Route {
	return []registry.Route{
		{Method: http.MethodGet, Pattern: "/config.json", Handler: p.handleConfig},
		{Method: http.MethodGet, Pattern: "/api/v1/crates", Handler: p.handleSearch},
		{Method: http.MethodPost, Pattern: "/api/v1/crates/new", Handler: p.handlePublish},
		{Method: http.MethodGet, Pattern: "/api/v1/crates/{crate}/{version}/download", Handler: p.handleDownload},
		// Sparse index: 1-char crate names
		{Method: http.MethodGet, Pattern: "/1/{crate}", Handler: p.handleIndex},
		// Sparse index: 2-char crate names
		{Method: http.MethodGet, Pattern: "/2/{crate}", Handler: p.handleIndex},
		// Sparse index: 3-char crate names
		{Method: http.MethodGet, Pattern: "/3/{prefix}/{crate}", Handler: p.handleIndex},
		// Sparse index: 4+ char crate names
		{Method: http.MethodGet, Pattern: "/{prefix1}/{prefix2}/{crate}", Handler: p.handleIndex},
	}
}

// --- Route Handlers ---

func (p *Plugin) handleConfig(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	upstream := p.config.Upstream
	p.mu.RUnlock()

	// Build base URL from request if possible, otherwise fall back to upstream.
	baseURL := upstream
	if r.Host != "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
			scheme = fwd
		}
		baseURL = fmt.Sprintf("%s://%s/cargo", scheme, r.Host)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"dl":  baseURL + "/api/v1/crates",
		"api": baseURL,
	})
}

func (p *Plugin) handleIndex(w http.ResponseWriter, r *http.Request) {
	crateName := extractParam(r, "crate")

	prefix := computePrefix(crateName)
	indexPath := fmt.Sprintf("cargo/index/%s/%s", prefix, strings.ToLower(crateName))

	reader, err := p.storage.Get(r.Context(), indexPath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"errors": []map[string]string{
				{"detail": "crate not found"},
			},
		})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (p *Plugin) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	// If no query, return empty results.
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"crates": []any{},
			"meta":   map[string]int{"total": 0},
		})
		return
	}

	// List all index entries and do a simple prefix/substring match.
	files, err := p.storage.List(r.Context(), "cargo/index")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"crates": []any{},
			"meta":   map[string]int{"total": 0},
		})
		return
	}

	type crateResult struct {
		Name       string `json:"name"`
		MaxVersion string `json:"max_version"`
		Desc       string `json:"description"`
	}

	var results []crateResult
	seen := make(map[string]bool)
	lowerQ := strings.ToLower(q)

	for _, f := range files {
		if f.IsDir {
			continue
		}
		// The last segment of the path is the crate name.
		parts := strings.Split(f.Path, "/")
		crateName := parts[len(parts)-1]

		if seen[crateName] {
			continue
		}
		if !strings.Contains(strings.ToLower(crateName), lowerQ) {
			continue
		}

		seen[crateName] = true

		// Read the index to find the latest version.
		reader, readErr := p.storage.Get(r.Context(), f.Path)
		if readErr != nil {
			continue
		}
		data, _ := io.ReadAll(reader)
		reader.Close()

		var latestVersion string
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			var entry indexEntry
			if json.Unmarshal([]byte(line), &entry) == nil {
				latestVersion = entry.Vers
			}
		}

		results = append(results, crateResult{
			Name:       crateName,
			MaxVersion: latestVersion,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"crates": results,
		"meta":   map[string]int{"total": len(results)},
	})
}

func (p *Plugin) handlePublish(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"errors": []map[string]string{
				{"detail": "failed to read request body"},
			},
		})
		return
	}

	// Cargo publish wire format:
	// 4 bytes LE: JSON metadata length
	// N bytes:    JSON metadata
	// 4 bytes LE: crate file length
	// M bytes:    crate file (.crate)

	if len(body) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"errors": []map[string]string{
				{"detail": "request body too short"},
			},
		})
		return
	}

	jsonLen := uint32(body[0]) | uint32(body[1])<<8 | uint32(body[2])<<16 | uint32(body[3])<<24
	if uint32(len(body)) < 4+jsonLen {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"errors": []map[string]string{
				{"detail": "metadata length exceeds body size"},
			},
		})
		return
	}

	jsonData := body[4 : 4+jsonLen]
	remaining := body[4+jsonLen:]

	var meta publishRequest
	if err := json.Unmarshal(jsonData, &meta); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"errors": []map[string]string{
				{"detail": "invalid JSON metadata"},
			},
		})
		return
	}

	if meta.Name == "" || meta.Vers == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"errors": []map[string]string{
				{"detail": "name and version are required"},
			},
		})
		return
	}

	// Parse crate file bytes.
	var crateData []byte
	if len(remaining) >= 4 {
		crateLen := uint32(remaining[0]) | uint32(remaining[1])<<8 | uint32(remaining[2])<<16 | uint32(remaining[3])<<24
		if uint32(len(remaining)) >= 4+crateLen {
			crateData = remaining[4 : 4+crateLen]
		}
	}

	if len(crateData) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"errors": []map[string]string{
				{"detail": "missing crate file data"},
			},
		})
		return
	}

	// Compute SHA-256 checksum of the crate file.
	hash := sha256.Sum256(crateData)
	cksum := hex.EncodeToString(hash[:])

	lowerName := strings.ToLower(meta.Name)

	// Store the .crate file.
	cratePath := fmt.Sprintf("cargo/crates/%s/%s.crate", lowerName, meta.Vers)
	if err := p.storage.Put(r.Context(), cratePath, bytes.NewReader(crateData)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"errors": []map[string]string{
				{"detail": "failed to store crate file"},
			},
		})
		return
	}

	// Build the index entry.
	deps := make([]indexDep, 0, len(meta.Deps))
	for _, d := range meta.Deps {
		dep := indexDep{
			Name:            d.Name,
			Req:             d.VersionReq,
			Features:        d.Features,
			Optional:        d.Optional,
			DefaultFeatures: d.DefaultFeatures,
			Target:          d.Target,
			Kind:            d.Kind,
			Registry:        d.Registry,
			Package:         d.ExplicitNameInToml,
		}
		if dep.Features == nil {
			dep.Features = []string{}
		}
		deps = append(deps, dep)
	}

	features := meta.Features
	if features == nil {
		features = make(map[string][]string)
	}

	entry := indexEntry{
		Name:     lowerName,
		Vers:     meta.Vers,
		Deps:     deps,
		Cksum:    cksum,
		Features: features,
		Yanked:   false,
		Links:    meta.Links,
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"errors": []map[string]string{
				{"detail": "failed to encode index entry"},
			},
		})
		return
	}

	// Append to the index file (JSON lines format).
	prefix := computePrefix(lowerName)
	indexPath := fmt.Sprintf("cargo/index/%s/%s", prefix, lowerName)

	var existing []byte
	reader, readErr := p.storage.Get(r.Context(), indexPath)
	if readErr == nil {
		existing, _ = io.ReadAll(reader)
		reader.Close()
	}

	var newIndex bytes.Buffer
	if len(existing) > 0 {
		newIndex.Write(existing)
		// Ensure trailing newline before appending.
		if existing[len(existing)-1] != '\n' {
			newIndex.WriteByte('\n')
		}
	}
	newIndex.Write(entryJSON)
	newIndex.WriteByte('\n')

	if err := p.storage.Put(r.Context(), indexPath, bytes.NewReader(newIndex.Bytes())); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"errors": []map[string]string{
				{"detail": "failed to update index"},
			},
		})
		return
	}

	p.logger.Info("cargo crate published",
		"name", lowerName,
		"version", meta.Vers,
		"size", len(crateData),
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"warnings": map[string][]string{
			"invalid_categories": {},
			"invalid_badges":     {},
			"other":              {},
		},
	})
}

func (p *Plugin) handleDownload(w http.ResponseWriter, r *http.Request) {
	crateName := strings.ToLower(extractParam(r, "crate"))
	version := extractParam(r, "version")

	cratePath := fmt.Sprintf("cargo/crates/%s/%s.crate", crateName, version)
	reader, err := p.storage.Get(r.Context(), cratePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"errors": []map[string]string{
				{"detail": fmt.Sprintf("crate %s@%s not found", crateName, version)},
			},
		})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/x-tar")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

// --- Compile-time interface check ---
var _ registry.RegistryPlugin = (*Plugin)(nil)

// buildPublishBody constructs the Cargo publish wire-format body from JSON metadata and crate bytes.
func buildPublishBody(meta, crateFile []byte) []byte {
	metaLen := uint32(len(meta))
	crateLen := uint32(len(crateFile))

	buf := make([]byte, 0, 4+len(meta)+4+len(crateFile))
	buf = append(buf,
		byte(metaLen), byte(metaLen>>8), byte(metaLen>>16), byte(metaLen>>24),
	)
	buf = append(buf, meta...)
	buf = append(buf,
		byte(crateLen), byte(crateLen>>8), byte(crateLen>>16), byte(crateLen>>24),
	)
	buf = append(buf, crateFile...)
	return buf
}
