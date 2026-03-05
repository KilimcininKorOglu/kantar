package maven

import (
	"encoding/xml"
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
	p.Configure(map[string]any{"upstream": "https://repo1.maven.org/maven2"})

	r := chi.NewRouter()
	r.Route("/maven", func(r chi.Router) {
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

func TestMavenDeployAndDownload(t *testing.T) {
	_, r := setupTestPlugin(t)

	jarContent := "fake-jar-content-bytes"
	deployPath := "/maven/com/example/mylib/1.0.0/mylib-1.0.0.jar"

	// Deploy artifact
	req := httptest.NewRequest(http.MethodPut, deployPath, strings.NewReader(jarContent))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("deploy expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Download artifact
	req2 := httptest.NewRequest(http.MethodGet, deployPath, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("download expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	got := w2.Body.String()
	if got != jarContent {
		t.Errorf("expected content %q, got %q", jarContent, got)
	}

	if ct := w2.Header().Get("Content-Type"); ct != "application/java-archive" {
		t.Errorf("expected Content-Type application/java-archive, got %s", ct)
	}
}

func TestMavenDeployPOM(t *testing.T) {
	_, r := setupTestPlugin(t)

	pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project>
  <groupId>com.example</groupId>
  <artifactId>mylib</artifactId>
  <version>1.0.0</version>
</project>`

	deployPath := "/maven/com/example/mylib/1.0.0/mylib-1.0.0.pom"

	req := httptest.NewRequest(http.MethodPut, deployPath, strings.NewReader(pomContent))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("deploy POM expected 201, got %d", w.Code)
	}

	// Download POM
	req2 := httptest.NewRequest(http.MethodGet, deployPath, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("download POM expected 200, got %d", w2.Code)
	}

	if ct := w2.Header().Get("Content-Type"); ct != "application/xml" {
		t.Errorf("expected Content-Type application/xml, got %s", ct)
	}
}

func TestMavenMetadataGeneration(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Deploy two versions
	for _, ver := range []string{"1.0.0", "2.0.0"} {
		path := "/maven/org/example/utils/" + ver + "/utils-" + ver + ".jar"
		req := httptest.NewRequest(http.MethodPut, path, strings.NewReader("jar-"+ver))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("deploy %s expected 201, got %d", ver, w.Code)
		}
	}

	// Request metadata
	req := httptest.NewRequest(http.MethodGet, "/maven/org/example/utils/maven-metadata.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("metadata expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/xml") {
		t.Errorf("expected XML content type, got %s", ct)
	}

	var meta mavenMetadata
	if err := xml.NewDecoder(w.Body).Decode(&meta); err != nil {
		t.Fatalf("failed to decode metadata XML: %v", err)
	}

	if meta.GroupID != "org.example" {
		t.Errorf("expected groupId org.example, got %s", meta.GroupID)
	}
	if meta.ArtifactID != "utils" {
		t.Errorf("expected artifactId utils, got %s", meta.ArtifactID)
	}
	if meta.Versioning.Latest != "2.0.0" {
		t.Errorf("expected latest 2.0.0, got %s", meta.Versioning.Latest)
	}

	versionMap := make(map[string]bool)
	for _, v := range meta.Versioning.Versions.Version {
		versionMap[v] = true
	}
	if !versionMap["1.0.0"] || !versionMap["2.0.0"] {
		t.Errorf("expected versions [1.0.0, 2.0.0], got %v", meta.Versioning.Versions.Version)
	}
}

func TestMavenMetadataNotFound(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/maven/com/nonexistent/lib/maven-metadata.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestMavenArtifactNotFound(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/maven/com/example/missing/1.0.0/missing-1.0.0.jar", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestParseMavenPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantGroup  string
		wantArtID  string
		wantVer    string
		wantFile   string
		wantMeta   bool
		wantErr    bool
	}{
		{
			name:      "artifact path",
			path:      "com/example/mylib/1.0.0/mylib-1.0.0.jar",
			wantGroup: "com.example",
			wantArtID: "mylib",
			wantVer:   "1.0.0",
			wantFile:  "mylib-1.0.0.jar",
		},
		{
			name:      "metadata path",
			path:      "com/example/mylib/maven-metadata.xml",
			wantGroup: "com.example",
			wantArtID: "mylib",
			wantMeta:  true,
		},
		{
			name:      "deep group path",
			path:      "org/apache/commons/commons-lang3/3.14.0/commons-lang3-3.14.0.jar",
			wantGroup: "org.apache.commons",
			wantArtID: "commons-lang3",
			wantVer:   "3.14.0",
			wantFile:  "commons-lang3-3.14.0.jar",
		},
		{
			name:      "POM file",
			path:      "io/netty/netty-all/4.1.100/netty-all-4.1.100.pom",
			wantGroup: "io.netty",
			wantArtID: "netty-all",
			wantVer:   "4.1.100",
			wantFile:  "netty-all-4.1.100.pom",
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "too short",
			path:    "com/mylib",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseMavenPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if info.groupID != tt.wantGroup {
				t.Errorf("groupID: want %q, got %q", tt.wantGroup, info.groupID)
			}
			if info.artifactID != tt.wantArtID {
				t.Errorf("artifactID: want %q, got %q", tt.wantArtID, info.artifactID)
			}
			if info.isMetadata != tt.wantMeta {
				t.Errorf("isMetadata: want %v, got %v", tt.wantMeta, info.isMetadata)
			}
			if !tt.wantMeta {
				if info.version != tt.wantVer {
					t.Errorf("version: want %q, got %q", tt.wantVer, info.version)
				}
				if info.filename != tt.wantFile {
					t.Errorf("filename: want %q, got %q", tt.wantFile, info.filename)
				}
			}
		})
	}
}

func TestGenerateMetadataXML(t *testing.T) {
	data, err := generateMetadataXML("com.example", "mylib", []string{"2.0.0", "1.0.0", "1.5.0"})
	if err != nil {
		t.Fatalf("generateMetadataXML failed: %v", err)
	}

	var meta mavenMetadata
	if err := xml.Unmarshal(data, &meta); err != nil {
		t.Fatalf("failed to unmarshal generated XML: %v", err)
	}

	if meta.GroupID != "com.example" {
		t.Errorf("groupId: want com.example, got %s", meta.GroupID)
	}
	if meta.ArtifactID != "mylib" {
		t.Errorf("artifactId: want mylib, got %s", meta.ArtifactID)
	}
	if meta.Versioning.Latest != "2.0.0" {
		t.Errorf("latest: want 2.0.0, got %s", meta.Versioning.Latest)
	}
	if meta.Versioning.Release != "2.0.0" {
		t.Errorf("release: want 2.0.0, got %s", meta.Versioning.Release)
	}

	// Versions should be sorted
	expectedOrder := []string{"1.0.0", "1.5.0", "2.0.0"}
	if len(meta.Versioning.Versions.Version) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(meta.Versioning.Versions.Version))
	}
	for i, v := range meta.Versioning.Versions.Version {
		if v != expectedOrder[i] {
			t.Errorf("version[%d]: want %s, got %s", i, expectedOrder[i], v)
		}
	}
}

func TestGenerateMetadataXMLEmpty(t *testing.T) {
	_, err := generateMetadataXML("com.example", "mylib", []string{})
	if err == nil {
		t.Error("expected error for empty version list, got nil")
	}
}

func TestMavenPluginInterface(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewFilesystem(dir)
	p := New(store, slog.Default())

	// Verify interface compliance
	var _ registry.RegistryPlugin = p

	if p.Name() != "Maven Repository" {
		t.Errorf("expected name 'Maven Repository', got %q", p.Name())
	}
	if p.Ecosystem() != registry.EcosystemMaven {
		t.Errorf("expected ecosystem %q, got %q", registry.EcosystemMaven, p.Ecosystem())
	}
	if p.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", p.Version())
	}

	routes := p.Routes()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}

	defaults := p.DefaultConfig()
	upstream, ok := defaults["upstream"].(string)
	if !ok || upstream != "https://repo1.maven.org/maven2" {
		t.Errorf("expected default upstream https://repo1.maven.org/maven2, got %v", defaults["upstream"])
	}

	// Validate
	result, err := p.ValidatePackage(nil, nil)
	if err != nil {
		t.Errorf("ValidatePackage returned error: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid result")
	}

	// FetchMetadata
	meta, err := p.FetchMetadata(nil, "com.example:mylib")
	if err != nil {
		t.Errorf("FetchMetadata returned error: %v", err)
	}
	if meta.Name != "com.example:mylib" {
		t.Errorf("expected name com.example:mylib, got %s", meta.Name)
	}
	if meta.Registry != registry.EcosystemMaven {
		t.Errorf("expected registry maven, got %s", meta.Registry)
	}
}

func TestMavenDeployMetadata(t *testing.T) {
	_, r := setupTestPlugin(t)

	metaContent := `<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <groupId>com.example</groupId>
  <artifactId>mylib</artifactId>
</metadata>`

	req := httptest.NewRequest(http.MethodPut, "/maven/com/example/mylib/maven-metadata.xml",
		strings.NewReader(metaContent))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("deploy metadata expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMavenContentTypes(t *testing.T) {
	_, r := setupTestPlugin(t)

	tests := []struct {
		filename    string
		contentType string
	}{
		{"mylib-1.0.0.jar", "application/java-archive"},
		{"mylib-1.0.0.pom", "application/xml"},
		{"mylib-1.0.0.jar.sha1", "text/plain"},
		{"mylib-1.0.0.jar.md5", "text/plain"},
		{"mylib-1.0.0.tar.gz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			deployPath := "/maven/com/example/mylib/1.0.0/" + tt.filename

			// Deploy
			req := httptest.NewRequest(http.MethodPut, deployPath, strings.NewReader("content"))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("deploy expected 201, got %d", w.Code)
			}

			// Download and check content type
			req2 := httptest.NewRequest(http.MethodGet, deployPath, nil)
			w2 := httptest.NewRecorder()
			r.ServeHTTP(w2, req2)
			if w2.Code != http.StatusOK {
				t.Fatalf("download expected 200, got %d", w2.Code)
			}

			if ct := w2.Header().Get("Content-Type"); ct != tt.contentType {
				t.Errorf("expected Content-Type %q, got %q", tt.contentType, ct)
			}
		})
	}
}

func TestMavenInvalidPath(t *testing.T) {
	_, r := setupTestPlugin(t)

	// GET with a path too short to be a valid Maven artifact or metadata
	req := httptest.NewRequest(http.MethodGet, "/maven/invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid path, got %d", w.Code)
	}
}

func TestMavenServePackage(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewFilesystem(dir)
	p := New(store, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/maven/anything", nil)
	w := httptest.NewRecorder()
	p.ServePackage(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("ServePackage expected 404, got %d", w.Code)
	}
}

func TestMavenSearch(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewFilesystem(dir)
	p := New(store, slog.Default())

	results, err := p.Search(nil, "anything")
	if err != nil {
		t.Errorf("Search returned error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

// Ensure io import is used in tests for coverage.
var _ = io.Discard
