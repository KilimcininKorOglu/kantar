// Package docker implements the Docker Registry API v2 plugin for Kantar.
package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/KilimcininKorOglu/kantar/internal/cache"
	"github.com/KilimcininKorOglu/kantar/internal/storage"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

// Plugin implements the RegistryPlugin interface for Docker images.
type Plugin struct {
	mu       sync.RWMutex
	storage  storage.Storage
	logger   *slog.Logger
	appCache cache.Cache
	config   pluginConfig
	upstream *upstreamClient
}

type pluginConfig struct {
	Upstream string `json:"upstream"`
}

// New creates a new Docker registry plugin.
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

func (p *Plugin) Name() string                      { return "Docker Registry" }
func (p *Plugin) Ecosystem() registry.EcosystemType { return registry.EcosystemDocker }

func (p *Plugin) Configure(config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if upstream, ok := config["upstream"].(string); ok {
		p.config.Upstream = upstream
	}
	if p.config.Upstream == "" {
		p.config.Upstream = "https://registry-1.docker.io"
	}

	username, _ := config["username"].(string)
	password, _ := config["password"].(string)

	p.upstream = newUpstreamClient(p.config.Upstream, username, password, p.logger)
	return nil
}

func (p *Plugin) DefaultConfig() map[string]any {
	return map[string]any{
		"upstream": "https://registry-1.docker.io",
	}
}

func (p *Plugin) Search(_ context.Context, query string) ([]registry.PackageMeta, error) {
	// Docker Hub search is limited; return empty for now
	return nil, nil
}

func (p *Plugin) FetchPackage(_ context.Context, name, version string) (*registry.Package, error) {
	return nil, fmt.Errorf("use FetchMetadata and pull manifest/blobs separately for Docker images")
}

func (p *Plugin) FetchMetadata(_ context.Context, name string) (*registry.PackageMeta, error) {
	return &registry.PackageMeta{
		Name:     name,
		Registry: registry.EcosystemDocker,
	}, nil
}

func (p *Plugin) ServePackage(w http.ResponseWriter, r *http.Request) {
	// Handled by individual route handlers
	http.NotFound(w, r)
}

func (p *Plugin) PublishPackage(_ context.Context, _ *registry.Package) error {
	return fmt.Errorf("use Docker push protocol (manifest + blob upload)")
}

func (p *Plugin) DeletePackage(_ context.Context, name, version string) error {
	return p.storage.Delete(context.Background(), fmt.Sprintf("docker/manifests/%s/%s", name, version))
}

func (p *Plugin) ValidatePackage(_ context.Context, pkg *registry.Package) (*registry.ValidationResult, error) {
	return &registry.ValidationResult{Valid: true}, nil
}

func (p *Plugin) Routes() []registry.Route {
	return []registry.Route{
		{Method: http.MethodGet, Pattern: "/", Handler: p.handleAPIVersionCheck},
		{Method: http.MethodGet, Pattern: "/{name}/manifests/{reference}", Handler: p.handleManifestGet},
		{Method: http.MethodPut, Pattern: "/{name}/manifests/{reference}", Handler: p.handleManifestPut},
		{Method: http.MethodHead, Pattern: "/{name}/manifests/{reference}", Handler: p.handleManifestHead},
		{Method: http.MethodDelete, Pattern: "/{name}/manifests/{reference}", Handler: p.handleManifestDelete},
		{Method: http.MethodGet, Pattern: "/{name}/blobs/{digest}", Handler: p.handleBlobGet},
		{Method: http.MethodHead, Pattern: "/{name}/blobs/{digest}", Handler: p.handleBlobHead},
		{Method: http.MethodPost, Pattern: "/{name}/blobs/uploads/", Handler: p.handleBlobUploadInit},
		{Method: http.MethodPatch, Pattern: "/{name}/blobs/uploads/{uuid}", Handler: p.handleBlobUploadChunk},
		{Method: http.MethodPut, Pattern: "/{name}/blobs/uploads/{uuid}", Handler: p.handleBlobUploadComplete},
		{Method: http.MethodGet, Pattern: "/{name}/tags/list", Handler: p.handleTagsList},
	}
}

// --- Route Handlers ---

func (p *Plugin) handleAPIVersionCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{})
}

func (p *Plugin) handleManifestGet(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r, "name")
	reference := extractPathParam(r, "reference")
	ctx := r.Context()

	path := fmt.Sprintf("docker/manifests/%s/%s", name, reference)
	reader, err := p.storage.Get(ctx, path)
	if err == nil {
		defer reader.Close()
		data, _ := io.ReadAll(reader)
		contentType := detectManifestMediaType(data)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Docker-Content-Digest", computeDigest(data))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	// Storage miss — proxy from upstream.
	if p.upstream == nil {
		writeError(w, http.StatusNotFound, "MANIFEST_UNKNOWN", "manifest not found")
		return
	}

	upstreamName := normalizeImageName(name)
	acceptHeader := r.Header.Get("Accept")

	p.logger.Debug("proxying manifest from upstream", "name", name, "ref", reference)

	body, contentType, digest, fetchErr := p.upstream.fetchManifest(ctx, upstreamName, reference, acceptHeader)
	if fetchErr != nil {
		p.logger.Warn("upstream manifest fetch failed", "name", name, "ref", reference, "error", fetchErr)
		writeError(w, http.StatusNotFound, "MANIFEST_UNKNOWN", "manifest not found")
		return
	}

	// Store manifest locally by the original reference (tag or digest).
	if storeErr := p.storage.Put(ctx, path, bytes.NewReader(body)); storeErr != nil {
		p.logger.Warn("failed to cache manifest", "path", path, "error", storeErr)
	}

	// If the reference is a tag and the digest is available, also store by digest
	// so subsequent blob/manifest pulls by digest hit local storage.
	if !strings.HasPrefix(reference, "sha256:") && digest != "" {
		digestPath := fmt.Sprintf("docker/manifests/%s/%s", name, digest)
		if storeErr := p.storage.Put(ctx, digestPath, bytes.NewReader(body)); storeErr != nil {
			p.logger.Warn("failed to cache manifest by digest", "path", digestPath, "error", storeErr)
		}
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func (p *Plugin) handleManifestHead(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r, "name")
	reference := extractPathParam(r, "reference")
	ctx := r.Context()

	path := fmt.Sprintf("docker/manifests/%s/%s", name, reference)
	reader, err := p.storage.Get(ctx, path)
	if err == nil {
		defer reader.Close()
		data, _ := io.ReadAll(reader)
		contentType := detectManifestMediaType(data)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Docker-Content-Digest", computeDigest(data))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Storage miss — proxy HEAD from upstream.
	if p.upstream == nil {
		writeError(w, http.StatusNotFound, "MANIFEST_UNKNOWN", "manifest not found")
		return
	}

	upstreamName := normalizeImageName(name)
	acceptHeader := r.Header.Get("Accept")

	body, contentType, digest, fetchErr := p.upstream.fetchManifest(ctx, upstreamName, reference, acceptHeader)
	if fetchErr != nil {
		writeError(w, http.StatusNotFound, "MANIFEST_UNKNOWN", "manifest not found")
		return
	}

	// Cache it locally for future requests.
	if storeErr := p.storage.Put(ctx, path, bytes.NewReader(body)); storeErr != nil {
		p.logger.Warn("failed to cache manifest on HEAD", "path", path, "error", storeErr)
	}

	if !strings.HasPrefix(reference, "sha256:") && digest != "" {
		digestPath := fmt.Sprintf("docker/manifests/%s/%s", name, digest)
		if storeErr := p.storage.Put(ctx, digestPath, bytes.NewReader(body)); storeErr != nil {
			p.logger.Warn("failed to cache manifest by digest on HEAD", "path", digestPath, "error", storeErr)
		}
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) handleManifestPut(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r, "name")
	reference := extractPathParam(r, "reference")

	path := fmt.Sprintf("docker/manifests/%s/%s", name, reference)
	if err := p.storage.Put(r.Context(), path, r.Body); err != nil {
		writeError(w, http.StatusInternalServerError, "MANIFEST_INVALID", "failed to store manifest")
		return
	}

	p.logger.Info("manifest stored", "name", name, "ref", reference)
	w.Header().Set("Location", fmt.Sprintf("/v2/%s/manifests/%s", name, reference))
	w.WriteHeader(http.StatusCreated)
}

func (p *Plugin) handleManifestDelete(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r, "name")
	reference := extractPathParam(r, "reference")

	path := fmt.Sprintf("docker/manifests/%s/%s", name, reference)
	if err := p.storage.Delete(r.Context(), path); err != nil {
		writeError(w, http.StatusNotFound, "MANIFEST_UNKNOWN", "manifest not found")
		return
	}

	w.WriteHeader(http.StatusAccepted)
	p.logger.Info("manifest deleted", "name", name, "ref", reference)
}

func (p *Plugin) handleBlobGet(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r, "name")
	digest := extractPathParam(r, "digest")
	ctx := r.Context()

	path := fmt.Sprintf("docker/blobs/%s/%s", name, digest)
	reader, err := p.storage.Get(ctx, path)
	if err == nil {
		defer reader.Close()
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Docker-Content-Digest", digest)
		w.WriteHeader(http.StatusOK)
		io.Copy(w, reader)
		return
	}

	// Storage miss — stream from upstream while caching simultaneously.
	if p.upstream == nil {
		writeError(w, http.StatusNotFound, "BLOB_UNKNOWN", "blob not found")
		return
	}

	upstreamName := normalizeImageName(name)

	p.logger.Debug("proxying blob from upstream", "name", name, "digest", digest)

	upstreamReader, size, fetchErr := p.upstream.fetchBlob(ctx, upstreamName, digest)
	if fetchErr != nil {
		p.logger.Warn("upstream blob fetch failed", "name", name, "digest", digest, "error", fetchErr)
		writeError(w, http.StatusNotFound, "BLOB_UNKNOWN", "blob not found")
		return
	}
	defer upstreamReader.Close()

	// Use io.Pipe + io.TeeReader to stream to both storage and client simultaneously.
	// This avoids buffering the entire blob in memory.
	blobPath := fmt.Sprintf("docker/blobs/%s/%s", name, digest)
	pr, pw := io.Pipe()

	// Storage write runs in a background goroutine.
	storeDone := make(chan error, 1)
	go func() {
		storeDone <- p.storage.Put(ctx, blobPath, pr)
	}()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Docker-Content-Digest", digest)
	if size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	w.WriteHeader(http.StatusOK)

	// TeeReader: everything read from upstream is also written into the pipe.
	_, copyErr := io.Copy(w, io.TeeReader(upstreamReader, pw))
	pw.Close()

	if storeErr := <-storeDone; storeErr != nil {
		p.logger.Warn("failed to cache blob", "path", blobPath, "error", storeErr)
	}
	if copyErr != nil {
		p.logger.Warn("error streaming blob to client", "name", name, "digest", digest, "error", copyErr)
	}
}

func (p *Plugin) handleBlobHead(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r, "name")
	digest := extractPathParam(r, "digest")
	ctx := r.Context()

	path := fmt.Sprintf("docker/blobs/%s/%s", name, digest)
	info, err := p.storage.Stat(ctx, path)
	if err == nil {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))
		w.Header().Set("Docker-Content-Digest", digest)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Storage miss — check upstream for existence/size.
	if p.upstream == nil {
		writeError(w, http.StatusNotFound, "BLOB_UNKNOWN", "blob not found")
		return
	}

	upstreamName := normalizeImageName(name)
	size, headErr := p.upstream.fetchBlobHead(ctx, upstreamName, digest)
	if headErr != nil {
		writeError(w, http.StatusNotFound, "BLOB_UNKNOWN", "blob not found")
		return
	}

	w.Header().Set("Docker-Content-Digest", digest)
	if size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) handleBlobUploadInit(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r, "name")

	uuid := generateUUID()
	uploadPath := fmt.Sprintf("docker/uploads/%s/%s", name, uuid)

	// If digest is provided, this is a monolithic upload
	if digest := r.URL.Query().Get("digest"); digest != "" {
		blobPath := fmt.Sprintf("docker/blobs/%s/%s", name, digest)
		if err := p.storage.Put(r.Context(), blobPath, r.Body); err != nil {
			writeError(w, http.StatusInternalServerError, "BLOB_UPLOAD_UNKNOWN", "upload failed")
			return
		}
		w.Header().Set("Docker-Content-Digest", digest)
		w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
		w.WriteHeader(http.StatusCreated)
		return
	}

	// Start chunked upload
	_ = uploadPath
	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid))
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.Header().Set("Range", "0-0")
	w.WriteHeader(http.StatusAccepted)
}

func (p *Plugin) handleBlobUploadChunk(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r, "name")
	uuid := extractPathParam(r, "uuid")

	uploadPath := fmt.Sprintf("docker/uploads/%s/%s", name, uuid)
	if err := p.storage.Put(r.Context(), uploadPath, r.Body); err != nil {
		writeError(w, http.StatusInternalServerError, "BLOB_UPLOAD_UNKNOWN", "chunk upload failed")
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid))
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.WriteHeader(http.StatusAccepted)
}

func (p *Plugin) handleBlobUploadComplete(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r, "name")
	uuid := extractPathParam(r, "uuid")
	digest := r.URL.Query().Get("digest")

	if digest == "" {
		writeError(w, http.StatusBadRequest, "DIGEST_INVALID", "digest parameter required")
		return
	}

	// Move from uploads to blobs
	uploadPath := fmt.Sprintf("docker/uploads/%s/%s", name, uuid)
	blobPath := fmt.Sprintf("docker/blobs/%s/%s", name, digest)

	// Read uploaded data and store as blob
	reader, err := p.storage.Get(r.Context(), uploadPath)
	if err != nil {
		// Maybe data was sent in this request body
		if err := p.storage.Put(r.Context(), blobPath, r.Body); err != nil {
			writeError(w, http.StatusInternalServerError, "BLOB_UPLOAD_UNKNOWN", "failed to complete upload")
			return
		}
	} else {
		defer reader.Close()
		if err := p.storage.Put(r.Context(), blobPath, reader); err != nil {
			writeError(w, http.StatusInternalServerError, "BLOB_UPLOAD_UNKNOWN", "failed to move blob")
			return
		}
		p.storage.Delete(r.Context(), uploadPath)
	}

	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/%s", name, digest))
	w.WriteHeader(http.StatusCreated)
}

func (p *Plugin) handleTagsList(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r, "name")
	ctx := r.Context()

	// Collect local tags.
	tagSet := make(map[string]struct{})

	prefix := fmt.Sprintf("docker/manifests/%s", name)
	files, err := p.storage.List(ctx, prefix)
	if err == nil {
		for _, f := range files {
			if !f.IsDir {
				parts := strings.Split(f.Path, "/")
				tag := parts[len(parts)-1]
				if !strings.HasPrefix(tag, "sha256:") {
					tagSet[tag] = struct{}{}
				}
			}
		}
	}

	// Merge upstream tags if available.
	if p.upstream != nil {
		upstreamName := normalizeImageName(name)
		upstreamTags, fetchErr := p.upstream.fetchTagsList(ctx, upstreamName)
		if fetchErr != nil {
			p.logger.Debug("upstream tags fetch failed", "name", name, "error", fetchErr)
		} else {
			for _, t := range upstreamTags {
				tagSet[t] = struct{}{}
			}
		}
	}

	tags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"name": name,
		"tags": tags,
	})
}
