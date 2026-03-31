package helm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
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
	p.Configure(map[string]any{})

	r := chi.NewRouter()
	r.Route("/helm", func(r chi.Router) {
		for _, route := range p.Routes() {
			switch route.Method {
			case http.MethodGet:
				r.Get(route.Pattern, route.Handler)
			case http.MethodPost:
				r.Post(route.Pattern, route.Handler)
			case http.MethodDelete:
				r.Delete(route.Pattern, route.Handler)
			}
		}
	})

	return p, r
}

func uploadChart(t *testing.T, router chi.Router, name, version, description string) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the chart file field with fake tgz data.
	part, err := writer.CreateFormFile("chart", fmt.Sprintf("%s-%s.tgz", name, version))
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write([]byte("fake-chart-tgz-content-" + name + "-" + version))

	writer.WriteField("name", name)
	writer.WriteField("version", version)
	if description != "" {
		writer.WriteField("description", description)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/helm/api/charts", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestUploadAndDownloadChart(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Upload a chart.
	w := uploadChart(t, r, "mychart", "0.1.0", "A test chart")
	if w.Code != http.StatusCreated {
		t.Fatalf("upload expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var uploadResp map[string]any
	json.NewDecoder(w.Body).Decode(&uploadResp)
	if uploadResp["saved"] != true {
		t.Errorf("expected saved=true, got %v", uploadResp["saved"])
	}

	// Download the chart.
	req := httptest.NewRequest(http.MethodGet, "/helm/charts/mychart-0.1.0.tgz", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Fatalf("download expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	if w2.Header().Get("Content-Type") != "application/gzip" {
		t.Errorf("expected Content-Type application/gzip, got %s", w2.Header().Get("Content-Type"))
	}

	body := w2.Body.String()
	if !strings.Contains(body, "fake-chart-tgz-content-mychart-0.1.0") {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestIndexYAMLGeneration(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Upload two charts.
	uploadChart(t, r, "nginx", "1.0.0", "Nginx chart")
	uploadChart(t, r, "nginx", "1.1.0", "Nginx chart updated")
	uploadChart(t, r, "redis", "5.0.0", "Redis chart")

	// Get the index.
	req := httptest.NewRequest(http.MethodGet, "/helm/index.yaml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("index expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if w.Header().Get("Content-Type") != "application/x-yaml" {
		t.Errorf("expected Content-Type application/x-yaml, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()

	// Verify YAML structure.
	if !strings.Contains(body, "apiVersion: v1") {
		t.Error("index.yaml missing apiVersion: v1")
	}
	if !strings.Contains(body, "entries:") {
		t.Error("index.yaml missing entries:")
	}
	if !strings.Contains(body, "nginx") {
		t.Error("index.yaml missing nginx chart")
	}
	if !strings.Contains(body, "redis") {
		t.Error("index.yaml missing redis chart")
	}
	if !strings.Contains(body, "version: 1.0.0") {
		t.Error("index.yaml missing version 1.0.0")
	}
	if !strings.Contains(body, "version: 1.1.0") {
		t.Error("index.yaml missing version 1.1.0")
	}
	if !strings.Contains(body, "version: 5.0.0") {
		t.Error("index.yaml missing version 5.0.0")
	}
	if !strings.Contains(body, "charts/nginx-1.0.0.tgz") {
		t.Error("index.yaml missing URL for nginx-1.0.0.tgz")
	}
}

func TestDeleteChart(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Upload two versions.
	uploadChart(t, r, "myapp", "1.0.0", "My app")
	uploadChart(t, r, "myapp", "2.0.0", "My app v2")

	// Delete version 1.0.0.
	req := httptest.NewRequest(http.MethodDelete, "/helm/api/charts/myapp/1.0.0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("delete expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify download of deleted version returns 404.
	req2 := httptest.NewRequest(http.MethodGet, "/helm/charts/myapp-1.0.0.tgz", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted chart, got %d", w2.Code)
	}

	// Verify version 2.0.0 still exists.
	req3 := httptest.NewRequest(http.MethodGet, "/helm/charts/myapp-2.0.0.tgz", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("expected 200 for remaining chart, got %d", w3.Code)
	}

	// Verify index only contains version 2.0.0.
	req4 := httptest.NewRequest(http.MethodGet, "/helm/index.yaml", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	indexBody := w4.Body.String()
	if strings.Contains(indexBody, "version: 1.0.0") {
		t.Error("deleted version 1.0.0 still in index")
	}
	if !strings.Contains(indexBody, "version: 2.0.0") {
		t.Error("remaining version 2.0.0 missing from index")
	}
}

func TestDeleteNonexistentChart(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodDelete, "/helm/api/charts/nonexistent/1.0.0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDownloadNonexistentChart(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/helm/charts/nope-1.0.0.tgz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestEmptyIndex(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/helm/index.yaml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "apiVersion: v1") {
		t.Error("empty index missing apiVersion")
	}
	if !strings.Contains(body, "entries:") {
		t.Error("empty index missing entries")
	}
}

func TestPluginInterface(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewFilesystem(dir)
	p := New(store, slog.Default())

	// Verify interface compliance at compile time.
	var _ registry.RegistryPlugin = p

	if p.Name() != "Helm Chart Registry" {
		t.Errorf("expected 'Helm Chart Registry', got '%s'", p.Name())
	}
	if p.Ecosystem() != registry.EcosystemHelm {
		t.Errorf("expected %s, got %s", registry.EcosystemHelm, p.Ecosystem())
	}
	if p.Version() != "0.1.0" {
		t.Errorf("expected 0.1.0, got %s", p.Version())
	}
	if len(p.Routes()) != 4 {
		t.Errorf("expected 4 routes, got %d", len(p.Routes()))
	}

	// Default config should have empty upstream.
	cfg := p.DefaultConfig()
	if cfg["upstream"] != "" {
		t.Errorf("expected empty upstream, got %v", cfg["upstream"])
	}

	// Search should return nil without error.
	results, err := p.Search(nil, "test")
	if err != nil {
		t.Errorf("search returned error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}

	// FetchPackage should return an error.
	_, err = p.FetchPackage(nil, "test", "1.0.0")
	if err == nil {
		t.Error("expected error from FetchPackage")
	}

	// ValidatePackage should return valid.
	result, err := p.ValidatePackage(nil, nil)
	if err != nil {
		t.Errorf("validate returned error: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid result")
	}

	// ServePackage should return 404.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	p.ServePackage(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// PublishPackage should return an error.
	err = p.PublishPackage(nil, nil)
	if err == nil {
		t.Error("expected error from PublishPackage")
	}

	// FetchMetadata should return basic metadata.
	meta, err := p.FetchMetadata(nil, "test-chart")
	if err != nil {
		t.Errorf("fetch metadata error: %v", err)
	}
	if meta.Name != "test-chart" {
		t.Errorf("expected name test-chart, got %s", meta.Name)
	}
	if meta.Registry != registry.EcosystemHelm {
		t.Errorf("expected helm registry, got %s", meta.Registry)
	}
}

func TestUploadMissingFields(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Upload without name field.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("chart", "test.tgz")
	part.Write([]byte("data"))
	writer.WriteField("version", "1.0.0")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/helm/api/charts", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", w.Code)
	}
}

func TestUploadMissingChartFile(t *testing.T) {
	_, r := setupTestPlugin(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("name", "test")
	writer.WriteField("version", "1.0.0")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/helm/api/charts", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing chart file, got %d", w.Code)
	}
}

func TestDeleteLastVersionRemovesMetadata(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Upload a single version.
	uploadChart(t, r, "single", "1.0.0", "Single version chart")

	// Delete it.
	req := httptest.NewRequest(http.MethodDelete, "/helm/api/charts/single/1.0.0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("delete expected 200, got %d", w.Code)
	}

	// Index should not contain the chart.
	req2 := httptest.NewRequest(http.MethodGet, "/helm/index.yaml", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	body := w2.Body.String()
	if strings.Contains(body, "single") {
		t.Error("deleted chart 'single' still appears in index")
	}
}

func TestVersionReplaceOnReupload(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Upload same version twice with different descriptions.
	uploadChart(t, r, "reup", "1.0.0", "First upload")
	uploadChart(t, r, "reup", "1.0.0", "Second upload")

	// Index should only have one entry for version 1.0.0.
	req := httptest.NewRequest(http.MethodGet, "/helm/index.yaml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	count := strings.Count(body, "version: 1.0.0")
	if count != 1 {
		t.Errorf("expected exactly 1 entry for version 1.0.0, found %d", count)
	}
}

func TestGenerateIndexYAMLHelperEmpty(t *testing.T) {
	yaml := generateIndexYAML(map[string][]chartMeta{})

	if !strings.Contains(yaml, "apiVersion: v1") {
		t.Error("missing apiVersion")
	}
	if !strings.Contains(yaml, "entries:") {
		t.Error("missing entries")
	}
	if !strings.Contains(yaml, "{}") {
		t.Error("empty entries should contain {}")
	}
}

func TestGenerateIndexYAMLHelperWithData(t *testing.T) {
	entries := map[string][]chartMeta{
		"app": {
			{
				Name:    "app",
				Version: "2.0.0",
				Digest:  "abc123",
				URLs:    []string{"charts/app-2.0.0.tgz"},
			},
		},
	}

	yaml := generateIndexYAML(entries)

	if !strings.Contains(yaml, "app:") {
		t.Error("missing chart name in YAML")
	}
	if !strings.Contains(yaml, "version: 2.0.0") {
		t.Error("missing version in YAML")
	}
	if !strings.Contains(yaml, "digest: abc123") {
		t.Error("missing digest in YAML")
	}
	if !strings.Contains(yaml, "charts/app-2.0.0.tgz") {
		t.Error("missing URL in YAML")
	}
}

// Ensure unused imports are consumed.
var _ = io.Discard
var _ = fmt.Sprint
