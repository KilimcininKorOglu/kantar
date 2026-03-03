package docker

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/storage"
)

func setupTestPlugin(t *testing.T) (*Plugin, chi.Router) {
	t.Helper()

	dir := t.TempDir()
	store, err := storage.NewFilesystem(dir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	p := New(store, slog.Default())
	p.Configure(map[string]any{"upstream": "https://registry-1.docker.io"})

	r := chi.NewRouter()
	r.Route("/v2", func(r chi.Router) {
		for _, route := range p.Routes() {
			switch route.Method {
			case http.MethodGet:
				r.Get(route.Pattern, route.Handler)
			case http.MethodPut:
				r.Put(route.Pattern, route.Handler)
			case http.MethodPost:
				r.Post(route.Pattern, route.Handler)
			case http.MethodDelete:
				r.Delete(route.Pattern, route.Handler)
			case http.MethodHead:
				r.Head(route.Pattern, route.Handler)
			case http.MethodPatch:
				r.Patch(route.Pattern, route.Handler)
			}
		}
	})

	return p, r
}

func TestAPIVersionCheck(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/v2/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	apiVersion := w.Header().Get("Docker-Distribution-API-Version")
	if apiVersion != "registry/2.0" {
		t.Errorf("expected registry/2.0, got %s", apiVersion)
	}
}

func TestManifestPutAndGet(t *testing.T) {
	_, r := setupTestPlugin(t)

	manifest := `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json"}`

	// PUT manifest
	req := httptest.NewRequest(http.MethodPut, "/v2/nginx/manifests/latest", bytes.NewReader([]byte(manifest)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("PUT expected 201, got %d", w.Code)
	}

	// GET manifest
	req2 := httptest.NewRequest(http.MethodGet, "/v2/nginx/manifests/latest", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("GET expected 200, got %d", w2.Code)
	}

	digest := w2.Header().Get("Docker-Content-Digest")
	if digest == "" || digest[:7] != "sha256:" {
		t.Errorf("expected sha256 digest, got %q", digest)
	}
}

func TestManifestNotFound(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/v2/nonexistent/manifests/latest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var errResp map[string]any
	json.NewDecoder(w.Body).Decode(&errResp)
	errors, ok := errResp["errors"].([]any)
	if !ok || len(errors) == 0 {
		t.Error("expected errors array in response")
	}
}

func TestBlobMonolithicUpload(t *testing.T) {
	_, r := setupTestPlugin(t)

	blobData := []byte("fake layer data")
	digest := computeDigest(blobData)

	// Monolithic upload
	req := httptest.NewRequest(http.MethodPost, "/v2/nginx/blobs/uploads/?digest="+digest, bytes.NewReader(blobData))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("upload expected 201, got %d", w.Code)
	}

	// GET blob
	req2 := httptest.NewRequest(http.MethodGet, "/v2/nginx/blobs/"+digest, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("GET blob expected 200, got %d", w2.Code)
	}

	if w2.Body.String() != string(blobData) {
		t.Errorf("blob content mismatch")
	}
}

func TestManifestDelete(t *testing.T) {
	_, r := setupTestPlugin(t)

	manifest := `{"schemaVersion":2}`

	// PUT then DELETE
	req := httptest.NewRequest(http.MethodPut, "/v2/myapp/manifests/v1.0", bytes.NewReader([]byte(manifest)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	req2 := httptest.NewRequest(http.MethodDelete, "/v2/myapp/manifests/v1.0", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusAccepted {
		t.Errorf("DELETE expected 202, got %d", w2.Code)
	}

	// Verify gone
	req3 := httptest.NewRequest(http.MethodGet, "/v2/myapp/manifests/v1.0", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w3.Code)
	}
}

func TestTagsList(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Store some manifests
	for _, tag := range []string{"latest", "v1.0", "v2.0"} {
		req := httptest.NewRequest(http.MethodPut, "/v2/myapp/manifests/"+tag, bytes.NewReader([]byte(`{"schemaVersion":2}`)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	// List tags
	req := httptest.NewRequest(http.MethodGet, "/v2/myapp/tags/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["name"] != "myapp" {
		t.Errorf("expected name myapp, got %v", resp["name"])
	}

	tags, ok := resp["tags"].([]any)
	if !ok || len(tags) != 3 {
		t.Errorf("expected 3 tags, got %v", resp["tags"])
	}
}

func TestPluginInterface(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewFilesystem(dir)
	p := New(store, slog.Default())

	if p.Name() != "Docker Registry" {
		t.Errorf("expected Docker Registry, got %s", p.Name())
	}
	if p.Ecosystem() != "docker" {
		t.Errorf("expected docker, got %s", p.Ecosystem())
	}
	if len(p.Routes()) != 11 {
		t.Errorf("expected 11 routes, got %d", len(p.Routes()))
	}
}

func TestComputeDigest(t *testing.T) {
	data := []byte("test data")
	digest := computeDigest(data)
	if digest[:7] != "sha256:" {
		t.Errorf("expected sha256: prefix, got %s", digest)
	}
	if len(digest) != 71 { // "sha256:" + 64 hex chars
		t.Errorf("expected 71 char digest, got %d", len(digest))
	}
}
