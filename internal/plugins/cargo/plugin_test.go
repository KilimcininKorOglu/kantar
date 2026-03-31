package cargo

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
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
	p.Configure(map[string]any{"upstream": "https://crates.io"})

	r := chi.NewRouter()
	r.Route("/cargo", func(r chi.Router) {
		for _, route := range p.Routes() {
			switch route.Method {
			case http.MethodGet:
				r.Get(route.Pattern, route.Handler)
			case http.MethodPost:
				r.Post(route.Pattern, route.Handler)
			}
		}
	})

	return p, r
}

func TestCargoConfigJSON(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/cargo/config.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var config map[string]string
	if err := json.NewDecoder(w.Body).Decode(&config); err != nil {
		t.Fatalf("decode config: %v", err)
	}

	if config["dl"] == "" {
		t.Error("config.dl should not be empty")
	}
	if config["api"] == "" {
		t.Error("config.api should not be empty")
	}
}

func TestCargoPublishAndDownload(t *testing.T) {
	_, r := setupTestPlugin(t)

	meta := publishRequest{
		Name: "my-crate",
		Vers: "0.1.0",
		Deps: []publishDep{
			{
				Name:            "serde",
				VersionReq:      "^1.0",
				Features:        []string{"derive"},
				Optional:        false,
				DefaultFeatures: true,
				Kind:            "normal",
			},
		},
		Features: map[string][]string{
			"default": {"std"},
			"std":     {},
		},
		License: "MIT",
		Desc:    "A test crate",
	}

	metaJSON, _ := json.Marshal(meta)
	crateContent := []byte("fake-crate-file-content-for-testing")
	body := buildPublishBody(metaJSON, crateContent)

	// Publish
	req := httptest.NewRequest(http.MethodPost, "/cargo/api/v1/crates/new", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("publish expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Download
	req2 := httptest.NewRequest(http.MethodGet, "/cargo/api/v1/crates/my-crate/0.1.0/download", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("download expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	downloaded, _ := io.ReadAll(w2.Body)
	if !bytes.Equal(downloaded, crateContent) {
		t.Errorf("downloaded content mismatch: got %q, want %q", downloaded, crateContent)
	}
}

func TestCargoPublishAndIndex(t *testing.T) {
	_, r := setupTestPlugin(t)

	meta := publishRequest{
		Name:    "tokio",
		Vers:    "1.0.0",
		Deps:    []publishDep{},
		Features: map[string][]string{"default": {}},
	}

	metaJSON, _ := json.Marshal(meta)
	crateContent := []byte("tokio-crate-bytes")
	body := buildPublishBody(metaJSON, crateContent)

	// Publish
	req := httptest.NewRequest(http.MethodPost, "/cargo/api/v1/crates/new", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("publish expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Read sparse index. "tokio" is 5 chars -> prefix "to/ki"
	req2 := httptest.NewRequest(http.MethodGet, "/cargo/to/ki/tokio", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("index expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Parse JSON lines
	lines := strings.Split(strings.TrimSpace(w2.Body.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 index entry, got %d", len(lines))
	}

	var entry indexEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("decode index entry: %v", err)
	}

	if entry.Name != "tokio" {
		t.Errorf("expected name tokio, got %s", entry.Name)
	}
	if entry.Vers != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", entry.Vers)
	}
	if entry.Cksum == "" {
		t.Error("checksum should not be empty")
	}
}

func TestComputePrefix(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"1-char crate", "a", "1"},
		{"2-char crate", "ab", "2"},
		{"3-char crate", "abc", "3/a"},
		{"3-char uppercase", "Abc", "3/a"},
		{"4-char crate", "abcd", "ab/cd"},
		{"5-char crate", "serde", "se/rd"},
		{"long crate", "tokio-util", "to/ki"},
		{"uppercase", "Tokio", "to/ki"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computePrefix(tc.input)
			if got != tc.expect {
				t.Errorf("computePrefix(%q) = %q, want %q", tc.input, got, tc.expect)
			}
		})
	}
}

func TestCargoSearch(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Publish a crate first so search has something to find.
	meta := publishRequest{
		Name:     "serde",
		Vers:     "1.0.0",
		Deps:     []publishDep{},
		Features: map[string][]string{},
	}
	metaJSON, _ := json.Marshal(meta)
	crateContent := []byte("serde-data")
	body := buildPublishBody(metaJSON, crateContent)

	pubReq := httptest.NewRequest(http.MethodPost, "/cargo/api/v1/crates/new", bytes.NewReader(body))
	pubW := httptest.NewRecorder()
	r.ServeHTTP(pubW, pubReq)

	if pubW.Code != http.StatusOK {
		t.Fatalf("publish expected 200, got %d", pubW.Code)
	}

	// Search for it.
	req := httptest.NewRequest(http.MethodGet, "/cargo/api/v1/crates?q=serde", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("search expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Crates []struct {
			Name       string `json:"name"`
			MaxVersion string `json:"max_version"`
		} `json:"crates"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode search result: %v", err)
	}

	if result.Meta.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Meta.Total)
	}
	if len(result.Crates) != 1 || result.Crates[0].Name != "serde" {
		t.Errorf("expected serde in results, got %+v", result.Crates)
	}
}

func TestCargoSearchEmpty(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/cargo/api/v1/crates?q=nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("search expected 200, got %d", w.Code)
	}
}

func TestCargoPluginInterface(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewFilesystem(dir)
	p := New(store, slog.Default())

	// Verify RegistryPlugin interface compliance.
	var _ registry.RegistryPlugin = p

	if p.Name() != "Cargo Registry" {
		t.Errorf("expected Cargo Registry, got %s", p.Name())
	}
	if p.Ecosystem() != registry.EcosystemCargo {
		t.Errorf("expected cargo, got %s", p.Ecosystem())
	}
	if p.Version() != "0.1.0" {
		t.Errorf("expected 0.1.0, got %s", p.Version())
	}

	routes := p.Routes()
	if len(routes) < 5 {
		t.Errorf("expected at least 5 routes, got %d", len(routes))
	}

	defaults := p.DefaultConfig()
	if defaults["upstream"] != "https://crates.io" {
		t.Errorf("expected default upstream https://crates.io, got %s", defaults["upstream"])
	}

	result, err := p.ValidatePackage(nil, nil)
	if err != nil {
		t.Errorf("ValidatePackage returned error: %v", err)
	}
	if !result.Valid {
		t.Error("ValidatePackage should return valid")
	}
}

func TestCargoDownloadNotFound(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/cargo/api/v1/crates/nonexistent/0.0.1/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCargoIndexNotFound(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/cargo/se/rd/serde", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCargoPublishMultipleVersions(t *testing.T) {
	_, r := setupTestPlugin(t)

	publishCrate := func(name, version string) {
		meta := publishRequest{
			Name:     name,
			Vers:     version,
			Deps:     []publishDep{},
			Features: map[string][]string{},
		}
		metaJSON, _ := json.Marshal(meta)
		crateContent := []byte("crate-" + version)
		body := buildPublishBody(metaJSON, crateContent)

		req := httptest.NewRequest(http.MethodPost, "/cargo/api/v1/crates/new", bytes.NewReader(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("publish %s@%s expected 200, got %d: %s", name, version, w.Code, w.Body.String())
		}
	}

	publishCrate("multi", "0.1.0")
	publishCrate("multi", "0.2.0")

	// Read index; "multi" is 5 chars -> prefix "mu/lt"
	req := httptest.NewRequest(http.MethodGet, "/cargo/mu/lt/multi", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("index expected 200, got %d", w.Code)
	}

	lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 index entries, got %d: %v", len(lines), lines)
	}

	var entry1, entry2 indexEntry
	json.Unmarshal([]byte(lines[0]), &entry1)
	json.Unmarshal([]byte(lines[1]), &entry2)

	if entry1.Vers != "0.1.0" {
		t.Errorf("first entry version = %s, want 0.1.0", entry1.Vers)
	}
	if entry2.Vers != "0.2.0" {
		t.Errorf("second entry version = %s, want 0.2.0", entry2.Vers)
	}
}

func TestCargoPublishInvalidBody(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Too-short body
	req := httptest.NewRequest(http.MethodPost, "/cargo/api/v1/crates/new", bytes.NewReader([]byte{0x01}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
