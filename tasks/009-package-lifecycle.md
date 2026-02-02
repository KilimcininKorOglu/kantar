# Feature 009: Package Lifecycle Management

**Feature ID:** F009
**Feature Name:** Package Lifecycle Management
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1.5 weeks
**Status:** NOT_STARTED

## Overview

Paket yaşam döngüsü yönetimi: upstream'den arama, onaylama/engelleme workflow'u, senkronizasyon, private paket yayınlama, versiyon yönetimi. Allowlist ve Mirror modlarının implementasyonu.

## Goals

- Allowlist mode: yalnızca onaylı paketler çekilebilir
- Mirror mode: tümü çekilir, blocklist ile engelleme
- Onay workflow'u: request → policy check → approve/block
- Upstream senkronizasyon (otomatik ve manuel)
- Private paket yayınlama desteği

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T038-T043)
- [ ] Allowlist modunda onaysız paket engellenebiliyor
- [ ] Mirror modunda blocklist çalışıyor
- [ ] Onay workflow'u doğru çalışıyor
- [ ] Upstream sync başarılı

## Tasks

### T038: Package Manager Core

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

Paket yönetim katmanı. Paket CRUD, versiyon yönetimi, durum takibi (pending, approved, blocked).

#### Technical Details

```go
type PackageStatus string
const (
    StatusPending  PackageStatus = "pending"
    StatusApproved PackageStatus = "approved"
    StatusBlocked  PackageStatus = "blocked"
)

type PackageManager struct {
    db      database.DB
    storage storage.Storage
    cache   cache.Cache
    plugins plugin.PluginRegistry
}

func (pm *PackageManager) GetPackage(ctx, registry, name) (*Package, error)
func (pm *PackageManager) ListPackages(ctx, registry, filter) ([]Package, error)
func (pm *PackageManager) ApprovePackage(ctx, registry, name, versions) error
func (pm *PackageManager) BlockPackage(ctx, registry, name, reason) error
```

#### Files to Touch

- `internal/manager/manager.go` (new)
- `internal/manager/types.go` (new)
- `internal/manager/manager_test.go` (new)

#### Dependencies

- T013 (DB query'leri), T026 (storage), T030 (cache), T033 (plugin interface)

#### Success Criteria

- [ ] Paket CRUD operasyonları çalışıyor
- [ ] Durum geçişleri doğru (pending → approved/blocked)
- [ ] Versiyon filtreleme çalışıyor (glob pattern)
- [ ] Unit test'ler yazıldı

---

### T039: Allowlist ve Mirror Mode

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

İki çalışma modunun implementasyonu. Allowlist'te yalnızca config'deki onaylı paketler çekilebilir; Mirror'da tümü akıyor, blocklist ile filtreleme.

#### Technical Details

Config'den mod okunacak:
```toml
[registry.npm]
mode = "allowlist"  # allowlist | mirror
```

Allowlist: glob pattern desteği (`@types/*`, `express@4.*`)
Mirror: blocklist ile engelleme

#### Files to Touch

- `internal/manager/allowlist.go` (new)
- `internal/manager/mirror.go` (new)
- `internal/manager/mode.go` (new)
- `internal/manager/allowlist_test.go` (new)

#### Dependencies

- T038, T006

#### Success Criteria

- [ ] Allowlist modunda onaysız paket erişiminde hata dönüyor
- [ ] Mirror modunda tüm paketler erişilebilir
- [ ] Blocklist'teki paketler mirror modunda engelleniyor
- [ ] Glob pattern matching çalışıyor
- [ ] Mod config'den okunuyor

---

### T040: Upstream Sync Engine

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

Upstream registry'lerden paket çekme mekanizması. Plugin'in FetchPackage/FetchMetadata metodlarını kullanarak paketleri indir, doğrula ve storage'a kaydet. Otomatik senkronizasyon (zamanlayıcı) ve manuel tetikleme.

#### Technical Details

Sync flow:
1. Plugin.FetchMetadata() — upstream'den metadata çek
2. Policy check — policy engine'den geçir
3. Plugin.FetchPackage() — paket dosyasını indir
4. Checksum verification — bütünlük kontrolü
5. Storage.Put() — lokal'e kaydet
6. DB update — kayıt oluştur

Auto-sync: `auto_sync_interval` config'e göre periyodik çalışma.

#### Files to Touch

- `internal/manager/sync.go` (new)
- `internal/manager/sync_test.go` (new)
- `internal/manager/scheduler.go` (new)

#### Dependencies

- T038, T027 (storage), T033 (plugin interface)

#### Success Criteria

- [ ] Tekil paket sync çalışıyor
- [ ] Toplu allowlist sync çalışıyor
- [ ] Checksum doğrulama yapılıyor
- [ ] Otomatik sync zamanlayıcısı çalışıyor
- [ ] Sync durumu ve ilerleme raporlanabiliyor

---

### T041: Onay Workflow'u

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Paket onay/engelleme API'si. Developer paket isteyebilir → admin onaylar/reddeder → onaylanan paket sync edilir.

#### Technical Details

```
POST /api/v1/registries/{type}/packages/{name}/request  — paket isteği
POST /api/v1/registries/{type}/packages/{name}/approve  — onayla
POST /api/v1/registries/{type}/packages/{name}/block    — engelle
GET  /api/v1/packages/pending                           — bekleyen istekler
```

#### Files to Touch

- `internal/server/handlers/packages.go` (new)
- `internal/manager/workflow.go` (new)

#### Dependencies

- T038, T025 (auth API)

#### Success Criteria

- [ ] Paket isteği oluşturuluyor (pending durumunda)
- [ ] Admin onay/ret işlemi çalışıyor
- [ ] Onaylanan paket otomatik sync ediliyor
- [ ] Bekleyen istekler listelenebiliyor
- [ ] Yetki kontrolü uygulanıyor (yalnızca admin onaylayabilir)

---

### T042: Package REST API

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

PRD Section 6.1'deki package management REST endpoint'leri. Listeleme, arama, detay, versiyon silme, bulk import/export.

#### Technical Details

```
GET    /api/v1/registries/{type}/packages          — listele (?search=&status=)
GET    /api/v1/registries/{type}/packages/{name}    — detay
DELETE /api/v1/registries/{type}/packages/{name}/{version} — versiyon sil
POST   /api/v1/registries/{type}/sync              — tam sync
POST   /api/v1/registries/{type}/sync/{name}       — tekil sync
POST   /api/v1/packages/import                     — toplu import
GET    /api/v1/packages/export                     — toplu export
```

#### Files to Touch

- `internal/server/handlers/packages.go` (update)
- `internal/server/handlers/sync.go` (new)
- `internal/server/handlers/import_export.go` (new)

#### Dependencies

- T038, T040, T041, T019

#### Success Criteria

- [ ] Tüm endpoint'ler çalışıyor
- [ ] Pagination doğru (limit/offset)
- [ ] Arama filtreleme çalışıyor
- [ ] Bulk import TOML formatında çalışıyor
- [ ] Export TOML/JSON formatında çalışıyor

---

### T043: Dependency Resolution

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Paket bağımlılık ağacını çözme ve onaylanmamış bağımlılıkları tespit etme. Recursive dependency check.

#### Files to Touch

- `internal/manager/deps.go` (new)
- `internal/manager/deps_test.go` (new)

#### Dependencies

- T038

#### Success Criteria

- [ ] Bağımlılık ağacı çıkarılabiliyor
- [ ] Onaylanmamış bağımlılıklar tespit ediliyor
- [ ] Döngüsel bağımlılıklar handle ediliyor
- [ ] Dependency tree formatında çıktı

## Risk Assessment

- **Orta Risk:** Upstream API'lerin farklılığı (her ecosystem farklı)
- **Çözüm:** Plugin interface soyutlaması bu karmaşıklığı gizliyor
- **Yüksek Risk:** Auto-sync altında upstream API rate limiting
- **Çözüm:** Retry with backoff, sync interval configurable
