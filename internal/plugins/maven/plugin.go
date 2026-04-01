// Package maven implements the Maven repository plugin for Kantar.
package maven

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/KilimcininKorOglu/kantar/internal/cache"
	"github.com/KilimcininKorOglu/kantar/internal/httpclient"
	"github.com/KilimcininKorOglu/kantar/internal/storage"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
	"github.com/go-chi/chi/v5"
)

const storagePrefix = "maven/repository"

// Plugin implements the RegistryPlugin interface for Maven artifacts.
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

// New creates a new Maven registry plugin.
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

func (p *Plugin) Name() string                      { return "Maven Repository" }
func (p *Plugin) Ecosystem() registry.EcosystemType { return registry.EcosystemMaven }

func (p *Plugin) Configure(config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if upstream, ok := config["upstream"].(string); ok {
		p.config.Upstream = upstream
	}
	if p.config.Upstream == "" {
		p.config.Upstream = "https://repo1.maven.org/maven2"
	}
	return nil
}

func (p *Plugin) DefaultConfig() map[string]any {
	return map[string]any{
		"upstream": "https://repo1.maven.org/maven2",
	}
}

func (p *Plugin) Search(_ context.Context, _ string) ([]registry.PackageMeta, error) {
	return nil, nil
}

func (p *Plugin) FetchPackage(_ context.Context, _, _ string) (*registry.Package, error) {
	return nil, fmt.Errorf("not implemented: use Maven deploy protocol")
}

func (p *Plugin) FetchMetadata(_ context.Context, name string) (*registry.PackageMeta, error) {
	return &registry.PackageMeta{
		Name:     name,
		Registry: registry.EcosystemMaven,
	}, nil
}

var mavenHTTPClient = httpclient.Shared

type pomProject struct {
	XMLName      xml.Name        `xml:"project"`
	Dependencies pomDependencies `xml:"dependencies"`
}

type pomDependencies struct {
	Dependency []pomDependency `xml:"dependency"`
}

type pomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   string `xml:"optional"`
}

// ResolveDependencies fetches the POM from upstream and parses <dependencies>.
// Maven name format: "groupId:artifactId" (e.g., "org.springframework:spring-core")
func (p *Plugin) ResolveDependencies(ctx context.Context, name, versionRange string) ([]registry.Dependency, string, error) {
	p.mu.RLock()
	upstream := p.config.Upstream
	p.mu.RUnlock()
	if upstream == "" {
		upstream = "https://repo1.maven.org/maven2"
	}

	parts := strings.SplitN(name, ":", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("invalid Maven coordinate: %s (expected groupId:artifactId)", name)
	}
	groupID, artifactID := parts[0], parts[1]
	groupPath := strings.ReplaceAll(groupID, ".", "/")

	version := versionRange
	if version == "" || version == "*" || version == "latest" {
		return nil, "", fmt.Errorf("maven requires explicit version for %s", name)
	}

	cacheKey := fmt.Sprintf("upstream:%s:%s@%s", p.Ecosystem(), name, version)

	var data []byte

	// Try cache first
	if p.appCache != nil {
		if cached, _ := p.appCache.Get(ctx, cacheKey); cached != nil {
			data = cached
		}
	}

	if data == nil {
		// Fetch POM from upstream
		url := fmt.Sprintf("%s/%s/%s/%s/%s-%s.pom", upstream, groupPath, artifactID, version, artifactID, version)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, "", err
		}

		resp, err := mavenHTTPClient.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("fetching POM for %s:%s: %w", name, version, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("upstream returned %d for %s POM", resp.StatusCode, name)
		}

		var readErr error
		data, readErr = io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, "", readErr
		}

		if p.appCache != nil {
			p.appCache.Set(ctx, cacheKey, data, 5*time.Minute)
		}
	}

	var pom pomProject
	if err := xml.Unmarshal(data, &pom); err != nil {
		return nil, "", fmt.Errorf("parsing POM XML for %s: %w", name, err)
	}

	var deps []registry.Dependency
	for _, d := range pom.Dependencies.Dependency {
		// Skip test and provided scope
		if d.Scope == "test" || d.Scope == "provided" || d.Scope == "system" {
			continue
		}
		optional := d.Optional == "true"
		deps = append(deps, registry.Dependency{
			Name:         d.GroupID + ":" + d.ArtifactID,
			VersionRange: d.Version,
			Optional:     optional,
		})
	}

	return deps, version, nil
}

// Ensure unused imports don't cause errors
var _ = path.Join

func (p *Plugin) ServePackage(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (p *Plugin) PublishPackage(_ context.Context, _ *registry.Package) error {
	return fmt.Errorf("use Maven deploy protocol")
}

func (p *Plugin) DeletePackage(_ context.Context, name, version string) error {
	groupPath := groupIDToPath(name)
	prefix := path.Join(storagePrefix, groupPath, version)
	return p.storage.Delete(context.Background(), prefix)
}

func (p *Plugin) ValidatePackage(_ context.Context, _ *registry.Package) (*registry.ValidationResult, error) {
	return &registry.ValidationResult{Valid: true}, nil
}

func (p *Plugin) Routes() []registry.Route {
	return []registry.Route{
		{Method: http.MethodGet, Pattern: "/*", Handler: p.handleGet},
		{Method: http.MethodPut, Pattern: "/*", Handler: p.handlePut},
	}
}

// --- Route Handlers ---

// handleGet serves artifact downloads and maven-metadata.xml requests.
func (p *Plugin) handleGet(w http.ResponseWriter, r *http.Request) {
	rawPath := chi.URLParam(r, "*")
	if rawPath == "" {
		writeErrorText(w, http.StatusBadRequest, "missing path")
		return
	}

	info, err := parseMavenPath(rawPath)
	if err != nil {
		p.logger.Debug("invalid maven path", "path", rawPath, "error", err)
		writeErrorText(w, http.StatusBadRequest, "invalid maven path")
		return
	}

	if info.isMetadata {
		p.serveMetadata(w, r, info)
		return
	}

	p.serveArtifact(w, r, info)
}

// handlePut deploys an artifact to the repository.
func (p *Plugin) handlePut(w http.ResponseWriter, r *http.Request) {
	rawPath := chi.URLParam(r, "*")
	if rawPath == "" {
		writeErrorText(w, http.StatusBadRequest, "missing path")
		return
	}

	info, err := parseMavenPath(rawPath)
	if err != nil {
		p.logger.Debug("invalid maven path for deploy", "path", rawPath, "error", err)
		writeErrorText(w, http.StatusBadRequest, "invalid maven path")
		return
	}

	if info.isMetadata {
		// Clients may upload maven-metadata.xml; we accept but regenerate on read.
		storagePath := p.artifactStoragePath(info.groupID, info.artifactID, "", "maven-metadata.xml")
		if err := p.storage.Put(r.Context(), storagePath, r.Body); err != nil {
			p.logger.Error("failed to store metadata", "path", storagePath, "error", err)
			writeErrorText(w, http.StatusInternalServerError, "failed to store metadata")
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	}

	storagePath := p.artifactStoragePath(info.groupID, info.artifactID, info.version, info.filename)

	if err := p.storage.Put(r.Context(), storagePath, r.Body); err != nil {
		p.logger.Error("failed to store artifact", "path", storagePath, "error", err)
		writeErrorText(w, http.StatusInternalServerError, "failed to store artifact")
		return
	}

	p.logger.Info("maven artifact deployed",
		"groupId", info.groupID,
		"artifactId", info.artifactID,
		"version", info.version,
		"filename", info.filename,
	)

	w.WriteHeader(http.StatusCreated)
}

// serveArtifact streams an artifact file from storage.
func (p *Plugin) serveArtifact(w http.ResponseWriter, r *http.Request, info *mavenPathInfo) {
	storagePath := p.artifactStoragePath(info.groupID, info.artifactID, info.version, info.filename)

	reader, err := p.storage.Get(r.Context(), storagePath)
	if err != nil {
		writeErrorText(w, http.StatusNotFound, "artifact not found")
		return
	}
	defer reader.Close()

	contentType := detectContentType(info.filename)
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

// serveMetadata generates and serves maven-metadata.xml by scanning stored versions.
func (p *Plugin) serveMetadata(w http.ResponseWriter, r *http.Request, info *mavenPathInfo) {
	artifactPrefix := path.Join(storagePrefix, groupIDToPath(info.groupID), info.artifactID)

	files, err := p.storage.List(r.Context(), artifactPrefix)
	if err != nil || len(files) == 0 {
		writeErrorText(w, http.StatusNotFound, "no versions found")
		return
	}

	versionSet := make(map[string]struct{})
	for _, f := range files {
		if f.IsDir {
			continue
		}
		// Extract version from path: {prefix}/{version}/{filename}
		rel := strings.TrimPrefix(f.Path, artifactPrefix)
		rel = strings.TrimPrefix(rel, "/")
		segments := strings.SplitN(rel, "/", 2)
		if len(segments) >= 2 && segments[0] != "" && segments[0] != "maven-metadata.xml" {
			versionSet[segments[0]] = struct{}{}
		}
	}

	if len(versionSet) == 0 {
		writeErrorText(w, http.StatusNotFound, "no versions found")
		return
	}

	versionList := make([]string, 0, len(versionSet))
	for v := range versionSet {
		versionList = append(versionList, v)
	}

	data, err := generateMetadataXML(info.groupID, info.artifactID, versionList)
	if err != nil {
		p.logger.Error("failed to generate metadata", "error", err)
		writeErrorText(w, http.StatusInternalServerError, "failed to generate metadata")
		return
	}

	writeXML(w, http.StatusOK, data)
}

// artifactStoragePath builds the storage path for a Maven artifact.
// For metadata without a version, pass version as "".
func (p *Plugin) artifactStoragePath(groupID, artifactID, version, filename string) string {
	groupPath := groupIDToPath(groupID)
	if version == "" {
		return path.Join(storagePrefix, groupPath, artifactID, filename)
	}
	return path.Join(storagePrefix, groupPath, artifactID, version, filename)
}

// detectContentType returns the MIME type based on the file extension.
func detectContentType(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".pom"):
		return "application/xml"
	case strings.HasSuffix(filename, ".jar"):
		return "application/java-archive"
	case strings.HasSuffix(filename, ".xml"):
		return "application/xml"
	case strings.HasSuffix(filename, ".sha1"):
		return "text/plain"
	case strings.HasSuffix(filename, ".sha256"):
		return "text/plain"
	case strings.HasSuffix(filename, ".sha512"):
		return "text/plain"
	case strings.HasSuffix(filename, ".md5"):
		return "text/plain"
	case strings.HasSuffix(filename, ".asc"):
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
