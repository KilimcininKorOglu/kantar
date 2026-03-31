package nuget

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/internal/storage"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

func setupTestPlugin(t *testing.T) (*Plugin, chi.Router) {
	t.Helper()

	dir := t.TempDir()
	store, err := storage.NewFilesystem(dir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	p := New(store, slog.Default())
	p.Configure(map[string]any{"upstream": "https://api.nuget.org"})

	r := chi.NewRouter()
	r.Route("/nuget", func(r chi.Router) {
		for _, route := range p.Routes() {
			switch route.Method {
			case http.MethodGet:
				r.Get(route.Pattern, route.Handler)
			case http.MethodPut:
				r.Put(route.Pattern, route.Handler)
			}
		}
	})

	return p, r
}

// pushPackage is a test helper that pushes a fake .nupkg to the registry.
func pushPackage(t *testing.T, router chi.Router, id, version string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("package", fmt.Sprintf("%s.%s.nupkg", id, version))
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write([]byte("PK-fake-nupkg-content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPut, "/nuget/v3/package", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-NuGet-Package-Id", id)
	req.Header.Set("X-NuGet-Package-Version", version)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("push expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServiceIndex(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/nuget/v3/index.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var idx serviceIndex
	if err := json.NewDecoder(w.Body).Decode(&idx); err != nil {
		t.Fatalf("decode service index: %v", err)
	}

	if idx.Version != "3.0.0" {
		t.Errorf("expected version 3.0.0, got %s", idx.Version)
	}
	if len(idx.Resources) != 3 {
		t.Errorf("expected 3 resources, got %d", len(idx.Resources))
	}
}

func TestPushAndDownload(t *testing.T) {
	_, r := setupTestPlugin(t)

	pushPackage(t, r, "Newtonsoft.Json", "13.0.1")

	// Download the package.
	url := "/nuget/v3/flatcontainer/newtonsoft.json/13.0.1/newtonsoft.json.13.0.1.nupkg"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("download expected 200, got %d: %s", w.Code, w.Body.String())
	}

	data, _ := io.ReadAll(w.Body)
	if !bytes.Contains(data, []byte("PK-fake-nupkg-content")) {
		t.Errorf("downloaded content does not match pushed content")
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/octet-stream" {
		t.Errorf("expected Content-Type application/octet-stream, got %s", ct)
	}
}

func TestVersionList(t *testing.T) {
	_, r := setupTestPlugin(t)

	pushPackage(t, r, "Serilog", "3.0.0")
	pushPackage(t, r, "Serilog", "3.1.0")
	// Push the same version again — should not duplicate.
	pushPackage(t, r, "Serilog", "3.1.0")

	req := httptest.NewRequest(http.MethodGet, "/nuget/v3/flatcontainer/serilog/index.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var vl versionList
	json.NewDecoder(w.Body).Decode(&vl)

	if len(vl.Versions) != 2 {
		t.Errorf("expected 2 versions, got %d: %v", len(vl.Versions), vl.Versions)
	}
}

func TestVersionListNotFound(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/nuget/v3/flatcontainer/nonexistent/index.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestSearch(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/nuget/v3/search?q=json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var sr searchResponse
	json.NewDecoder(w.Body).Decode(&sr)
	if sr.TotalHits != 0 {
		t.Errorf("expected 0 hits, got %d", sr.TotalHits)
	}
}

func TestPluginInterface(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewFilesystem(dir)
	p := New(store, slog.Default())

	// Verify RegistryPlugin compliance.
	var _ registry.RegistryPlugin = p

	if p.Name() != "NuGet Registry" {
		t.Errorf("expected NuGet Registry, got %s", p.Name())
	}
	if p.Ecosystem() != registry.EcosystemNuGet {
		t.Errorf("expected nuget, got %s", p.Ecosystem())
	}
	if len(p.Routes()) != 5 {
		t.Errorf("expected 5 routes, got %d", len(p.Routes()))
	}

	defaults := p.DefaultConfig()
	if defaults["upstream"] != "https://api.nuget.org" {
		t.Errorf("expected default upstream https://api.nuget.org, got %s", defaults["upstream"])
	}
}

func TestDownloadInvalidFilename(t *testing.T) {
	_, r := setupTestPlugin(t)

	pushPackage(t, r, "TestPkg", "1.0.0")

	// Request with wrong filename.
	req := httptest.NewRequest(
		http.MethodGet,
		"/nuget/v3/flatcontainer/testpkg/1.0.0/wrong-name.nupkg",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid filename, got %d", w.Code)
	}
}

func TestPushMissingHeaders(t *testing.T) {
	_, r := setupTestPlugin(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("package", "test.nupkg")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPut, "/nuget/v3/package", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Intentionally omit X-NuGet-Package-Id and X-NuGet-Package-Version.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing headers, got %d", w.Code)
	}
}
