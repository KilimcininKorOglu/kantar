package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KilimcininKorOglu/kantar/internal/config"
)

func newTestServer() *Server {
	cfg := config.ServerConfig{
		Host: "127.0.0.1",
		Port: 0,
	}
	logger := slog.Default()
	return New(cfg, logger, Dependencies{})
}

func TestHealthz(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestSystemStatusRequiresAuth(t *testing.T) {
	s := newTestServer() // Dependencies{} has nil JWTManager
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	// With nil JWTManager, authenticated routes should return 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when JWTManager is nil, got %d", w.Code)
	}
}

func TestNotFound(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestMountPluginRoutes(t *testing.T) {
	s := newTestServer()

	// Mount a simple handler under /test/
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plugin response"))
	})
	s.MountPluginRoutes("/test", mux)

	req := httptest.NewRequest(http.MethodGet, "/test/", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "plugin response" {
		t.Errorf("expected 'plugin response', got %q", w.Body.String())
	}
}
