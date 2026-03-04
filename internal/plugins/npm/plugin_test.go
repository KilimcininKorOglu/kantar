package npm

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
	p.Configure(map[string]any{"upstream": "https://registry.npmjs.org"})

	r := chi.NewRouter()
	r.Route("/npm", func(r chi.Router) {
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

func TestNpmPublishAndGet(t *testing.T) {
	_, r := setupTestPlugin(t)

	packument := Packument{
		Name:        "my-pkg",
		Description: "A test package",
		DistTags:    map[string]string{"latest": "1.0.0"},
		Versions: map[string]VersionDoc{
			"1.0.0": {
				Name:    "my-pkg",
				Version: "1.0.0",
				Dist: DistInfo{
					Tarball: "http://localhost/npm/my-pkg/-/my-pkg-1.0.0.tgz",
					Shasum:  "abc123",
				},
			},
		},
	}

	body, _ := json.Marshal(packument)

	// Publish
	req := httptest.NewRequest(http.MethodPut, "/npm/my-pkg", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("publish expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Get packument
	req2 := httptest.NewRequest(http.MethodGet, "/npm/my-pkg", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("get expected 200, got %d", w2.Code)
	}

	var got Packument
	json.NewDecoder(w2.Body).Decode(&got)
	if got.Name != "my-pkg" {
		t.Errorf("expected name my-pkg, got %s", got.Name)
	}
	if got.DistTags["latest"] != "1.0.0" {
		t.Errorf("expected latest 1.0.0, got %s", got.DistTags["latest"])
	}
}

func TestNpmGetVersion(t *testing.T) {
	_, r := setupTestPlugin(t)

	packument := Packument{
		Name: "express",
		Versions: map[string]VersionDoc{
			"4.18.2": {Name: "express", Version: "4.18.2", License: "MIT"},
			"5.0.0":  {Name: "express", Version: "5.0.0", License: "MIT"},
		},
	}

	body, _ := json.Marshal(packument)
	req := httptest.NewRequest(http.MethodPut, "/npm/express", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Get specific version
	req2 := httptest.NewRequest(http.MethodGet, "/npm/express/4.18.2", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}

	var vDoc VersionDoc
	json.NewDecoder(w2.Body).Decode(&vDoc)
	if vDoc.Version != "4.18.2" {
		t.Errorf("expected 4.18.2, got %s", vDoc.Version)
	}
}

func TestNpmGetNotFound(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/npm/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestNpmSearch(t *testing.T) {
	_, r := setupTestPlugin(t)

	req := httptest.NewRequest(http.MethodGet, "/npm/-/v1/search?text=express", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNpmPluginInterface(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewFilesystem(dir)
	p := New(store, slog.Default())

	if p.Name() != "npm Registry" {
		t.Errorf("expected npm Registry, got %s", p.Name())
	}
	if p.Ecosystem() != "npm" {
		t.Errorf("expected npm, got %s", p.Ecosystem())
	}
	if len(p.Routes()) != 5 {
		t.Errorf("expected 5 routes, got %d", len(p.Routes()))
	}
}
