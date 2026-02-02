# Feature 008: Plugin Architecture

**Feature ID:** F008
**Feature Name:** Plugin Architecture
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Kantar'ın kalbi. `RegistryPlugin` interface'i, plugin registry, plugin lifecycle yönetimi, route mounting, config delegation. Her ekosistem plugin'i bu interface'i implement eder. Compile-time plugin'ler (dinamik .so yükleme yok).

## Goals

- Temiz ve genişletilebilir plugin interface'i
- Plugin lifecycle yönetimi (init, configure, start, stop)
- Plugin route'larını core router'a mount etme
- Plugin bazlı config delegation

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T033-T037)
- [ ] RegistryPlugin interface'i tanımlı ve belgelenmiş
- [ ] Plugin'ler register edilebiliyor
- [ ] Plugin route'ları doğru mount ediliyor
- [ ] Plugin config'leri doğru delege ediliyor

## Tasks

### T033: RegistryPlugin Interface

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

PRD Section 4.2'deki RegistryPlugin interface'inin Go tanımı. Metadata, upstream operations, local registry operations, validation, configuration, routes.

#### Technical Details

```go
type EcosystemType string
const (
    EcosystemDocker  EcosystemType = "docker"
    EcosystemNPM     EcosystemType = "npm"
    EcosystemPyPI    EcosystemType = "pypi"
    EcosystemGoMod   EcosystemType = "gomod"
    EcosystemCargo   EcosystemType = "cargo"
    EcosystemMaven   EcosystemType = "maven"
    EcosystemNuGet   EcosystemType = "nuget"
    EcosystemHelm    EcosystemType = "helm"
)

type RegistryPlugin interface {
    Name() string
    Version() string
    EcosystemType() EcosystemType

    Search(ctx context.Context, query string) ([]PackageMeta, error)
    FetchPackage(ctx context.Context, name, version string) (*Package, error)
    FetchMetadata(ctx context.Context, name string) (*PackageMeta, error)

    ServePackage(w http.ResponseWriter, r *http.Request)
    PublishPackage(ctx context.Context, pkg *Package) error
    DeletePackage(ctx context.Context, name, version string) error

    ValidatePackage(ctx context.Context, pkg *Package) (*ValidationResult, error)

    Configure(config map[string]interface{}) error
    DefaultConfig() map[string]interface{}
    Routes() []Route
}
```

#### Files to Touch

- `pkg/registry/plugin.go` (new)
- `pkg/registry/types.go` (new)
- `pkg/registry/route.go` (new)
- `pkg/registry/errors.go` (new)

#### Dependencies

- T001

#### Success Criteria

- [ ] Interface tam ve belgelenmiş
- [ ] Tüm yardımcı tipler tanımlı (PackageMeta, Package, ValidationResult, Route)
- [ ] Error tipleri tanımlı
- [ ] Public API olarak `pkg/` altında

---

### T034: Plugin Registry (Kayıt Defteri)

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Plugin'lerin register edildiği ve yönetildiği merkezi kayıt defteri. Plugin ekleme, kaldırma, listeleme, tip ile arama.

#### Technical Details

```go
type PluginRegistry struct {
    plugins map[EcosystemType]RegistryPlugin
}

func (r *PluginRegistry) Register(plugin RegistryPlugin) error
func (r *PluginRegistry) Get(ecosystem EcosystemType) (RegistryPlugin, error)
func (r *PluginRegistry) List() []RegistryPlugin
func (r *PluginRegistry) Has(ecosystem EcosystemType) bool
```

#### Files to Touch

- `internal/plugin/registry.go` (new)
- `internal/plugin/registry_test.go` (new)

#### Dependencies

- T033

#### Success Criteria

- [ ] Plugin register/get/list çalışıyor
- [ ] Duplicate ecosystem register'da hata
- [ ] Thread-safe erişim
- [ ] Plugin bulunamadığında anlaşılır hata

---

### T035: Plugin Lifecycle Manager

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Plugin'lerin yaşam döngüsü yönetimi: initialization, configuration, start, stop. Config'den plugin-specific ayarları okuma ve plugin'e delege etme.

#### Files to Touch

- `internal/plugin/lifecycle.go` (new)
- `internal/plugin/lifecycle_test.go` (new)

#### Dependencies

- T034, T006 (config — registry config'leri)

#### Success Criteria

- [ ] Plugin'ler sıralı olarak initialize ediliyor
- [ ] Config plugin'e doğru delege ediliyor
- [ ] Hatalı config'de plugin başlamıyor ve anlaşılır hata
- [ ] Graceful shutdown'da tüm plugin'ler düzgün kapanıyor

---

### T036: Plugin Route Binder

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 day

#### Description

Plugin'lerin Routes() ile döndürdüğü route tanımlarını chi router'a mount etme. Her plugin kendi path prefix'i altında.

#### Files to Touch

- `internal/plugin/binder.go` (new)

#### Dependencies

- T034, T019 (route mounting altyapısı)

#### Success Criteria

- [ ] Plugin route'ları doğru prefix altında mount ediliyor
- [ ] Route conflict'leri tespit ediliyor
- [ ] Middleware zinciri plugin route'larına uygulanıyor

---

### T037: Built-in Plugin Registrasyonu

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 day

#### Description

Tüm built-in plugin'leri (Docker, npm, PyPI, Go Modules, Cargo, Maven, NuGet, Helm) başlangıçta register eden bootstrap kodu. Plugin'ler compile-time olarak binary'ye dahil.

#### Files to Touch

- `internal/plugin/builtin.go` (new)
- `cmd/kantar/plugins.go` (new)

#### Dependencies

- T034, ve en az bir plugin implementasyonu (F012-F019)

#### Success Criteria

- [ ] Tüm built-in plugin'ler register ediliyor
- [ ] Plugin sayısı ve isimleri loglanıyor
- [ ] Devre dışı bırakılan plugin'ler skip ediliyor (config ile)

## Risk Assessment

- **Orta Risk:** Interface tasarımı tüm ekosistem ihtiyaçlarını karşılamalı
- **Çözüm:** İlk iki plugin (npm + Docker) implementasyonu sırasında interface revize edilebilir
- **Dikkat:** Interface bir kez public API olduktan sonra breaking change maliyetli
