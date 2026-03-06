package pypi

import (
	"bytes"
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
	p.Configure(map[string]any{"upstream": "https://pypi.org"})

	r := chi.NewRouter()
	r.Route("/pypi", func(r chi.Router) {
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

// createUploadRequest builds a multipart form request mimicking twine upload.
func createUploadRequest(t *testing.T, name, version, filename string, content []byte) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writer.WriteField("name", name)
	writer.WriteField("version", version)
	writer.WriteField("filetype", "sdist")
	writer.WriteField("sha256_digest", "abcdef1234567890")

	part, err := writer.CreateFormFile("content", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write(content)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/pypi/", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestPyPIUploadAndDownload(t *testing.T) {
	_, r := setupTestPlugin(t)

	fileContent := []byte("fake-tar-gz-content")
	filename := "my-package-1.0.0.tar.gz"
	req := createUploadRequest(t, "my_package", "1.0.0", filename, fileContent)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("upload expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Download the file.
	req2 := httptest.NewRequest(http.MethodGet, "/pypi/packages/my-package/"+filename, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("download expected 200, got %d", w2.Code)
	}

	got, _ := io.ReadAll(w2.Body)
	if !bytes.Equal(got, fileContent) {
		t.Errorf("downloaded content mismatch: got %q, want %q", got, fileContent)
	}
}

func TestPyPISimpleIndex(t *testing.T) {
	_, r := setupTestPlugin(t)

	// Upload two packages.
	r.ServeHTTP(httptest.NewRecorder(), createUploadRequest(t, "alpha", "1.0.0", "alpha-1.0.0.tar.gz", []byte("a")))
	r.ServeHTTP(httptest.NewRecorder(), createUploadRequest(t, "beta", "2.0.0", "beta-2.0.0.tar.gz", []byte("b")))

	req := httptest.NewRequest(http.MethodGet, "/pypi/simple/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("simple index expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `<a href="/pypi/simple/alpha/">alpha</a>`) {
		t.Errorf("simple index missing alpha link, body: %s", body)
	}
	if !strings.Contains(body, `<a href="/pypi/simple/beta/">beta</a>`) {
		t.Errorf("simple index missing beta link, body: %s", body)
	}
}

func TestPyPISimplePackagePage(t *testing.T) {
	_, r := setupTestPlugin(t)

	filename := "my-package-1.0.0.tar.gz"
	r.ServeHTTP(httptest.NewRecorder(), createUploadRequest(t, "my_package", "1.0.0", filename, []byte("content")))

	req := httptest.NewRequest(http.MethodGet, "/pypi/simple/my-package/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("package page expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, filename) {
		t.Errorf("package page missing filename link, body: %s", body)
	}
	if !strings.Contains(body, "#sha256=abcdef1234567890") {
		t.Errorf("package page missing sha256 fragment, body: %s", body)
	}
}

func TestPyPIPackageNotFound(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/pypi/simple/nonexistent/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPyPIDownloadNotFound(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/pypi/packages/no-such-pkg/file.tar.gz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPyPIPluginInterface(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewFilesystem(dir)
	p := New(store, slog.Default())

	// Verify the plugin satisfies the RegistryPlugin interface.
	var _ registry.RegistryPlugin = p

	if p.Name() != "PyPI Registry" {
		t.Errorf("expected PyPI Registry, got %s", p.Name())
	}
	if p.Ecosystem() != registry.EcosystemPyPI {
		t.Errorf("expected pypi, got %s", p.Ecosystem())
	}
	if len(p.Routes()) != 4 {
		t.Errorf("expected 4 routes, got %d", len(p.Routes()))
	}
}

func TestNormalizePkgName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My_Package", "my-package"},
		{"some.dotted.name", "some-dotted-name"},
		{"UPPER-CASE", "upper-case"},
		{"already-normalized", "already-normalized"},
		{"mixed_Case.and-Hyphens", "mixed-case-and-hyphens"},
		{"multi___underscores", "multi-underscores"},
	}

	for _, tt := range tests {
		got := normalizePkgName(tt.input)
		if got != tt.want {
			t.Errorf("normalizePkgName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
