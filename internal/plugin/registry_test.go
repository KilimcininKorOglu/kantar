package plugin

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/KilimcininKorOglu/kantar/pkg/registry"
)

// mockPlugin is a minimal RegistryPlugin implementation for testing.
type mockPlugin struct {
	name      string
	version   string
	ecosystem registry.EcosystemType
	routes    []registry.Route
}

func (m *mockPlugin) Name() string                        { return m.name }
func (m *mockPlugin) Version() string                     { return m.version }
func (m *mockPlugin) Ecosystem() registry.EcosystemType   { return m.ecosystem }
func (m *mockPlugin) Search(_ context.Context, _ string) ([]registry.PackageMeta, error) {
	return nil, nil
}
func (m *mockPlugin) FetchPackage(_ context.Context, _, _ string) (*registry.Package, error) {
	return nil, nil
}
func (m *mockPlugin) FetchMetadata(_ context.Context, _ string) (*registry.PackageMeta, error) {
	return nil, nil
}
func (m *mockPlugin) ServePackage(_ http.ResponseWriter, _ *http.Request) {}
func (m *mockPlugin) PublishPackage(_ context.Context, _ *registry.Package) error {
	return nil
}
func (m *mockPlugin) DeletePackage(_ context.Context, _, _ string) error { return nil }
func (m *mockPlugin) ValidatePackage(_ context.Context, _ *registry.Package) (*registry.ValidationResult, error) {
	return &registry.ValidationResult{Valid: true}, nil
}
func (m *mockPlugin) Configure(_ map[string]any) error         { return nil }
func (m *mockPlugin) DefaultConfig() map[string]any            { return map[string]any{} }
func (m *mockPlugin) Routes() []registry.Route                 { return m.routes }

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry(slog.Default())

	p := &mockPlugin{name: "npm", version: "1.0", ecosystem: registry.EcosystemNPM}
	if err := reg.Register(p); err != nil {
		t.Fatalf("register: %v", err)
	}

	got, err := reg.Get(registry.EcosystemNPM)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name() != "npm" {
		t.Errorf("expected npm, got %s", got.Name())
	}
}

func TestRegistryDuplicateRegister(t *testing.T) {
	reg := NewRegistry(slog.Default())

	p1 := &mockPlugin{name: "npm1", ecosystem: registry.EcosystemNPM}
	p2 := &mockPlugin{name: "npm2", ecosystem: registry.EcosystemNPM}

	reg.Register(p1)
	err := reg.Register(p2)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	reg := NewRegistry(slog.Default())
	_, err := reg.Get(registry.EcosystemDocker)
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry(slog.Default())
	reg.Register(&mockPlugin{name: "npm", ecosystem: registry.EcosystemNPM})
	reg.Register(&mockPlugin{name: "docker", ecosystem: registry.EcosystemDocker})

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(list))
	}
}

func TestRegistryHas(t *testing.T) {
	reg := NewRegistry(slog.Default())
	reg.Register(&mockPlugin{name: "npm", ecosystem: registry.EcosystemNPM})

	if !reg.Has(registry.EcosystemNPM) {
		t.Error("expected Has to return true for npm")
	}
	if reg.Has(registry.EcosystemDocker) {
		t.Error("expected Has to return false for docker")
	}
}

func TestRegistryMountRoutes(t *testing.T) {
	reg := NewRegistry(slog.Default())

	p := &mockPlugin{
		name:      "npm",
		ecosystem: registry.EcosystemNPM,
		routes: []registry.Route{
			{
				Method:  http.MethodGet,
				Pattern: "/test",
				Handler: func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("npm plugin"))
				},
			},
		},
	}
	reg.Register(p)

	r := chi.NewRouter()
	reg.MountRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/npm/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "npm plugin" {
		t.Errorf("expected 'npm plugin', got %q", w.Body.String())
	}
}
