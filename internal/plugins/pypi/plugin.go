// Package pypi implements the PyPI registry API plugin for Kantar.
package pypi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/storage"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

// Plugin implements the RegistryPlugin interface for PyPI packages.
type Plugin struct {
	mu      sync.RWMutex
	storage storage.Storage
	logger  *slog.Logger
	config  pluginConfig
}

type pluginConfig struct {
	Upstream string `json:"upstream"`
}

// packageMeta is the internal metadata structure persisted for each package.
type packageMeta struct {
	Name     string               `json:"name"`
	Versions map[string]versionMeta `json:"versions"`
}

// versionMeta holds per-version metadata.
type versionMeta struct {
	Version      string `json:"version"`
	Filename     string `json:"filename"`
	Filetype     string `json:"filetype"`
	SHA256Digest string `json:"sha256Digest"`
}

// New creates a new PyPI registry plugin.
func New(store storage.Storage, logger *slog.Logger) *Plugin {
	return &Plugin{
		storage: store,
		logger:  logger,
	}
}

func (p *Plugin) Name() string                      { return "PyPI Registry" }
func (p *Plugin) Version() string                   { return "1.0.0" }
func (p *Plugin) Ecosystem() registry.EcosystemType { return registry.EcosystemPyPI }

func (p *Plugin) Configure(config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if upstream, ok := config["upstream"].(string); ok {
		p.config.Upstream = upstream
	}
	if p.config.Upstream == "" {
		p.config.Upstream = "https://pypi.org"
	}
	return nil
}

func (p *Plugin) DefaultConfig() map[string]any {
	return map[string]any{
		"upstream": "https://pypi.org",
	}
}

func (p *Plugin) Search(_ context.Context, _ string) ([]registry.PackageMeta, error) {
	return nil, nil
}

func (p *Plugin) FetchPackage(_ context.Context, _, _ string) (*registry.Package, error) {
	return nil, fmt.Errorf("not implemented: use upstream sync")
}

func (p *Plugin) FetchMetadata(_ context.Context, name string) (*registry.PackageMeta, error) {
	return &registry.PackageMeta{
		Name:     name,
		Registry: registry.EcosystemPyPI,
	}, nil
}

func (p *Plugin) ServePackage(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (p *Plugin) PublishPackage(_ context.Context, _ *registry.Package) error {
	return fmt.Errorf("use twine upload protocol")
}

func (p *Plugin) DeletePackage(_ context.Context, name, version string) error {
	normalized := normalizePkgName(name)
	return p.storage.Delete(context.Background(), fmt.Sprintf("pypi/packages/%s/%s", normalized, version))
}

func (p *Plugin) ValidatePackage(_ context.Context, _ *registry.Package) (*registry.ValidationResult, error) {
	return &registry.ValidationResult{Valid: true}, nil
}

func (p *Plugin) Routes() []registry.Route {
	return []registry.Route{
		{Method: http.MethodGet, Pattern: "/simple/", Handler: p.handleSimpleIndex},
		{Method: http.MethodGet, Pattern: "/simple/{package}/", Handler: p.handleSimplePackage},
		{Method: http.MethodGet, Pattern: "/packages/*", Handler: p.handleDownload},
		{Method: http.MethodPost, Pattern: "/", Handler: p.handleUpload},
	}
}

// --- Route Handlers ---

// handleSimpleIndex serves the PEP 503 simple index listing all known packages.
func (p *Plugin) handleSimpleIndex(w http.ResponseWriter, r *http.Request) {
	files, err := p.storage.List(r.Context(), "pypi/metadata")
	if err != nil {
		writeHTML(w, http.StatusOK, simplePageHTML("Simple Index", ""))
		return
	}

	var names []string
	for _, f := range files {
		if f.IsDir {
			continue
		}
		base := f.Path
		// Extract just the filename from the path.
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		if strings.HasSuffix(base, ".json") {
			name := strings.TrimSuffix(base, ".json")
			names = append(names, name)
		}
	}

	sort.Strings(names)

	var links strings.Builder
	for _, name := range names {
		fmt.Fprintf(&links, "<a href=\"/pypi/simple/%s/\">%s</a>\n", name, name)
	}

	writeHTML(w, http.StatusOK, simplePageHTML("Simple Index", links.String()))
}

// handleSimplePackage serves the PEP 503 package page listing all files for a package.
func (p *Plugin) handleSimplePackage(w http.ResponseWriter, r *http.Request) {
	name := extractParam(r, "package")
	normalized := normalizePkgName(name)

	meta, err := p.loadMeta(r.Context(), normalized)
	if err != nil {
		writeHTML(w, http.StatusNotFound, simplePageHTML("Not Found", "<p>Package not found.</p>"))
		return
	}

	// Sort versions for deterministic output.
	versions := make([]string, 0, len(meta.Versions))
	for v := range meta.Versions {
		versions = append(versions, v)
	}
	sort.Strings(versions)

	var links strings.Builder
	for _, v := range versions {
		vm := meta.Versions[v]
		href := fmt.Sprintf("/pypi/packages/%s/%s", normalized, vm.Filename)
		if vm.SHA256Digest != "" {
			href += "#sha256=" + vm.SHA256Digest
		}
		fmt.Fprintf(&links, "<a href=\"%s\">%s</a>\n", href, vm.Filename)
	}

	writeHTML(w, http.StatusOK, simplePageHTML("Links for "+meta.Name, links.String()))
}

// handleDownload serves a package file.
func (p *Plugin) handleDownload(w http.ResponseWriter, r *http.Request) {
	// The wildcard captures everything after /packages/
	path := chi.URLParam(r, "*")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	storagePath := "pypi/packages/" + path
	reader, err := p.storage.Get(r.Context(), storagePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

// handleUpload handles twine-compatible multipart form uploads.
func (p *Plugin) handleUpload(w http.ResponseWriter, r *http.Request) {
	// 32 MiB max memory for multipart parsing.
	const maxMemory = 32 << 20
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	version := r.FormValue("version")
	filetype := r.FormValue("filetype")
	sha256Digest := r.FormValue("sha256_digest")

	if name == "" || version == "" {
		http.Error(w, "name and version are required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("content")
	if err != nil {
		http.Error(w, "content file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	normalized := normalizePkgName(name)
	filename := header.Filename

	// Store the package file.
	pkgPath := fmt.Sprintf("pypi/packages/%s/%s", normalized, filename)
	fileData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read uploaded file", http.StatusInternalServerError)
		return
	}

	if err := p.storage.Put(r.Context(), pkgPath, bytes.NewReader(fileData)); err != nil {
		http.Error(w, "failed to store package", http.StatusInternalServerError)
		return
	}

	// Update metadata.
	p.mu.Lock()
	defer p.mu.Unlock()

	meta, _ := p.loadMeta(r.Context(), normalized)
	if meta == nil {
		meta = &packageMeta{
			Name:     name,
			Versions: make(map[string]versionMeta),
		}
	}

	meta.Versions[version] = versionMeta{
		Version:      version,
		Filename:     filename,
		Filetype:     filetype,
		SHA256Digest: sha256Digest,
	}

	if err := p.saveMeta(r.Context(), normalized, meta); err != nil {
		http.Error(w, "failed to store metadata", http.StatusInternalServerError)
		return
	}

	p.logger.Info("pypi package uploaded", "name", name, "version", version, "filename", filename)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, "OK")
}

// --- Metadata persistence ---

func (p *Plugin) metaPath(normalized string) string {
	return fmt.Sprintf("pypi/metadata/%s.json", normalized)
}

func (p *Plugin) loadMeta(ctx context.Context, normalized string) (*packageMeta, error) {
	reader, err := p.storage.Get(ctx, p.metaPath(normalized))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var meta packageMeta
	if err := json.NewDecoder(reader).Decode(&meta); err != nil {
		return nil, fmt.Errorf("decoding metadata: %w", err)
	}
	return &meta, nil
}

func (p *Plugin) saveMeta(ctx context.Context, normalized string, meta *packageMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("encoding metadata: %w", err)
	}
	return p.storage.Put(ctx, p.metaPath(normalized), bytes.NewReader(data))
}

// simplePageHTML returns a minimal PEP 503-compliant HTML page.
func simplePageHTML(title, body string) string {
	return "<!DOCTYPE html>\n<html><head><title>" + title + "</title></head><body>\n" + body + "</body></html>"
}
