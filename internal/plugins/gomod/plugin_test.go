package gomod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	p.Configure(map[string]any{"upstream": "https://proxy.golang.org"})

	r := chi.NewRouter()
	r.Route("/go", func(r chi.Router) {
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

// uploadArtifact is a test helper that PUTs an artifact to the router.
func uploadArtifact(t *testing.T, router chi.Router, path string, body []byte) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("upload %s: expected 201, got %d: %s", path, w.Code, w.Body.String())
	}
}

func TestStoreAndRetrieveInfo(t *testing.T) {
	_, r := setupTestPlugin(t)

	info := VersionInfo{
		Version: "v1.2.3",
		Time:    time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
	}
	body, _ := json.Marshal(info)

	uploadArtifact(t, r, "/go/github.com/example/mod/@v/v1.2.3.info", body)

	// Retrieve it.
	req := httptest.NewRequest(http.MethodGet, "/go/github.com/example/mod/@v/v1.2.3.info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got VersionInfo
	json.NewDecoder(w.Body).Decode(&got)
	if got.Version != "v1.2.3" {
		t.Errorf("expected version v1.2.3, got %s", got.Version)
	}
}

func TestStoreAndRetrieveMod(t *testing.T) {
	_, r := setupTestPlugin(t)

	modContent := []byte("module github.com/example/mod\n\ngo 1.21\n")
	uploadArtifact(t, r, "/go/github.com/example/mod/@v/v1.0.0.mod", modContent)

	req := httptest.NewRequest(http.MethodGet, "/go/github.com/example/mod/@v/v1.0.0.mod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("expected text/plain content type, got %s", ct)
	}
	if w.Body.String() != string(modContent) {
		t.Errorf("mod content mismatch: got %q", w.Body.String())
	}
}

func TestStoreAndRetrieveZip(t *testing.T) {
	_, r := setupTestPlugin(t)

	zipData := []byte("fake-zip-content-for-test")
	uploadArtifact(t, r, "/go/github.com/example/mod/@v/v1.0.0.zip", zipData)

	req := httptest.NewRequest(http.MethodGet, "/go/github.com/example/mod/@v/v1.0.0.zip", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("expected application/zip, got %s", ct)
	}
	if w.Body.String() != string(zipData) {
		t.Errorf("zip content mismatch")
	}
}

func TestVersionList(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Upload two versions (only .info triggers version list update).
	for _, v := range []string{"v1.0.0", "v1.1.0"} {
		info := VersionInfo{Version: v, Time: time.Now().UTC()}
		body, _ := json.Marshal(info)
		uploadArtifact(t, r, fmt.Sprintf("/go/github.com/example/mod/@v/%s.info", v), body)
	}

	req := httptest.NewRequest(http.MethodGet, "/go/github.com/example/mod/@v/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !containsLine(body, "v1.0.0") {
		t.Errorf("version list missing v1.0.0: %q", body)
	}
	if !containsLine(body, "v1.1.0") {
		t.Errorf("version list missing v1.1.0: %q", body)
	}
}

func TestVersionListEmpty(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/go/github.com/example/nonexistent/@v/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for empty list, got %d", w.Code)
	}
}

func TestLatestEndpoint(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Upload two versions.
	for _, v := range []string{"v1.0.0", "v2.0.0"} {
		info := VersionInfo{Version: v, Time: time.Now().UTC()}
		body, _ := json.Marshal(info)
		uploadArtifact(t, r, fmt.Sprintf("/go/github.com/example/mod/@v/%s.info", v), body)
	}

	req := httptest.NewRequest(http.MethodGet, "/go/github.com/example/mod/@latest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got VersionInfo
	json.NewDecoder(w.Body).Decode(&got)
	if got.Version != "v2.0.0" {
		t.Errorf("expected latest v2.0.0, got %s", got.Version)
	}
}

func TestLatestNotFound(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/go/github.com/example/nonexistent/@latest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestModulePathEncoding(t *testing.T) {
	tests := []struct {
		original string
		encoded  string
	}{
		{"github.com/Azure/go-sdk", "github.com/!azure/go-sdk"},
		{"github.com/example/mod", "github.com/example/mod"},
		{"github.com/GoogleCloudPlatform/k8s", "github.com/!google!cloud!platform/k8s"},
		{"lowercase/only", "lowercase/only"},
	}

	for _, tt := range tests {
		t.Run(tt.original, func(t *testing.T) {
			got := encodePath(tt.original)
			if got != tt.encoded {
				t.Errorf("encodePath(%q) = %q, want %q", tt.original, got, tt.encoded)
			}

			decoded, ok := decodePath(tt.encoded)
			if !ok {
				t.Fatalf("decodePath(%q) failed", tt.encoded)
			}
			if decoded != tt.original {
				t.Errorf("decodePath(%q) = %q, want %q", tt.encoded, decoded, tt.original)
			}
		})
	}
}

func TestDecodePathInvalid(t *testing.T) {
	// Trailing '!' with no following character.
	_, ok := decodePath("github.com/bad!")
	if ok {
		t.Error("expected decodePath to fail for trailing '!'")
	}

	// '!' followed by uppercase (invalid per spec).
	_, ok = decodePath("github.com/!A")
	if ok {
		t.Error("expected decodePath to fail for '!A'")
	}
}

func TestModulePathEncodingRoundTrip(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Module with uppercase letters that need encoding.
	info := VersionInfo{Version: "v0.1.0", Time: time.Now().UTC()}
	body, _ := json.Marshal(info)

	// Upload using the original module path (chi will route correctly).
	uploadArtifact(t, r, "/go/github.com/Azure/azure-sdk/@v/v0.1.0.info", body)

	// Retrieve using the same path.
	req := httptest.NewRequest(http.MethodGet, "/go/github.com/Azure/azure-sdk/@v/v0.1.0.info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got VersionInfo
	json.NewDecoder(w.Body).Decode(&got)
	if got.Version != "v0.1.0" {
		t.Errorf("expected v0.1.0, got %s", got.Version)
	}
}

func TestNotFoundVersionInfo(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/go/github.com/example/mod/@v/v9.9.9.info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPluginInterface(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewFilesystem(dir)
	p := New(store, slog.Default())

	// Verify it satisfies the RegistryPlugin interface.
	var _ registry.RegistryPlugin = p

	if p.Name() != "Go Module Proxy" {
		t.Errorf("expected 'Go Module Proxy', got %s", p.Name())
	}
	if p.Ecosystem() != registry.EcosystemGoMod {
		t.Errorf("expected gomod, got %s", p.Ecosystem())
	}
	if p.Version() != "1.0.0" {
		t.Errorf("expected 1.0.0, got %s", p.Version())
	}
	if len(p.Routes()) != 2 {
		t.Errorf("expected 2 routes, got %d", len(p.Routes()))
	}

	cfg := p.DefaultConfig()
	if cfg["upstream"] != "https://proxy.golang.org" {
		t.Errorf("unexpected default upstream: %v", cfg["upstream"])
	}
}

func TestDuplicateVersionNotAdded(t *testing.T) {
	_, r := setupTestPlugin(t)

	info := VersionInfo{Version: "v1.0.0", Time: time.Now().UTC()}
	body, _ := json.Marshal(info)

	// Upload the same version twice.
	uploadArtifact(t, r, "/go/github.com/example/dup/@v/v1.0.0.info", body)
	uploadArtifact(t, r, "/go/github.com/example/dup/@v/v1.0.0.info", body)

	req := httptest.NewRequest(http.MethodGet, "/go/github.com/example/dup/@v/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	lines := nonEmptyLines(w.Body.String())
	if len(lines) != 1 {
		t.Errorf("expected 1 version entry, got %d: %v", len(lines), lines)
	}
}

// --- helpers ---

func containsLine(text, line string) bool {
	for _, l := range nonEmptyLines(text) {
		if l == line {
			return true
		}
	}
	return false
}

func nonEmptyLines(text string) []string {
	var result []string
	raw, _ := io.ReadAll(bytes.NewReader([]byte(text)))
	for _, line := range bytes.Split(raw, []byte("\n")) {
		s := string(bytes.TrimSpace(line))
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
