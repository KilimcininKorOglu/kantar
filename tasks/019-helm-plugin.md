# Feature 019: Helm Chart Plugin

**Feature ID:** F019
**Feature Name:** Helm Chart Plugin
**Priority:** P2 - HIGH
**Target Version:** v1.0.0
**Estimated Duration:** 0.5 week
**Status:** NOT_STARTED

## Overview

Helm Chart Repository API uyumlu plugin. `helm install`, `helm push` desteği. OCI-based ve legacy index.yaml tabanlı repository desteği.

## Goals

- Helm chart repository uyumluluğu
- `helm repo add kantar` çalışmalı
- Chart push/pull desteği

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T083-T085)
- [ ] `helm install` çalışıyor
- [ ] `helm push` çalışıyor
- [ ] index.yaml doğru üretiliyor

## Tasks

### T083: Helm Repository Index ve Chart Serve

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Helm chart repository endpoint'leri: index.yaml ve chart tarball serve etme.

#### Technical Details

```
GET  /helm/index.yaml                    — chart index
GET  /helm/charts/{chart}-{version}.tgz  — chart tarball
POST /helm/api/charts                    — chart upload
DELETE /helm/api/charts/{name}/{version} — chart silme
```

index.yaml: Tüm chart'ların metadata'sını içeren YAML dosyası.

#### Files to Touch

- `internal/plugins/helm/plugin.go` (new)
- `internal/plugins/helm/routes.go` (new)
- `internal/plugins/helm/index.go` (new)
- `internal/plugins/helm/serve.go` (new)

#### Dependencies

- T033, T019

#### Success Criteria

- [ ] index.yaml doğru formatında serve ediliyor
- [ ] Chart tarball download çalışıyor
- [ ] `helm repo add` başarılı
- [ ] `helm search` çalışıyor

---

### T084: Chart Upload ve Upstream Sync

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Helm chart push ve upstream repository sync.

#### Files to Touch

- `internal/plugins/helm/upload.go` (new)
- `internal/plugins/helm/upstream.go` (new)

#### Dependencies

- T083, T027, T040

#### Success Criteria

- [ ] Chart upload çalışıyor
- [ ] index.yaml otomatik güncelleniyor
- [ ] Upstream sync çalışıyor

---

### T085: Helm Plugin Config ve Entegrasyon

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

Helm plugin tam entegrasyonu.

#### Files to Touch

- `internal/plugins/helm/config.go` (new)
- `internal/plugins/helm/plugin.go` (update)

#### Dependencies

- T083-T084

#### Success Criteria

- [ ] RegistryPlugin interface tam implement edildi
- [ ] End-to-end helm install/push çalışıyor

## Risk Assessment

- **Düşük Risk:** Helm chart repo API basit (index.yaml + tarball serve)
- **Dikkat:** OCI-based Helm chart'lar Docker Registry API v2 üzerinden çalışır — F012 ile paylaşılabilir
