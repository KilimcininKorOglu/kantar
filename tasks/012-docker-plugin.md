# Feature 012: Docker Registry Plugin

**Feature ID:** F012
**Feature Name:** Docker Registry Plugin
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1.5 weeks
**Status:** NOT_STARTED

## Overview

Docker Registry HTTP API v2 uyumlu plugin. Image pull/push, manifest yönetimi, blob storage, tag listeleme. Docker Hub upstream desteği. OCI Image Specification uyumlu.

## Goals

- Docker Registry API v2 tam uyumluluk
- `docker pull/push kantar.local:8080/image:tag` çalışmalı
- Docker Hub'dan allowlist image'ları sync
- OCI layout ile storage

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T054-T058)
- [ ] `docker pull` ile image indirilebiliyor
- [ ] `docker push` ile image yüklenebiliyor
- [ ] Docker Hub sync çalışıyor

## Tasks

### T054: Docker Registry API v2 — Base

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Docker Registry API v2 temel endpoint'leri: version check, authentication challenge.

#### Technical Details

```
GET /v2/                          — API version check (200 OK)
GET /v2/_catalog                  — repository listesi
GET /v2/{name}/tags/list          — tag listesi
```

Docker auth: `401 Unauthorized` → `WWW-Authenticate: Bearer realm="..."` → token → retry

#### Files to Touch

- `internal/plugins/docker/plugin.go` (new)
- `internal/plugins/docker/routes.go` (new)
- `internal/plugins/docker/auth.go` (new)

#### Dependencies

- T033 (RegistryPlugin interface), T019 (route mounting)

#### Success Criteria

- [ ] `/v2/` endpoint 200 dönüyor
- [ ] Auth challenge doğru header ile çalışıyor
- [ ] Catalog ve tag list endpoint'leri çalışıyor

---

### T055: Manifest Operations

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

Image manifest GET/PUT/DELETE. Manifest schema v2 ve OCI manifest desteği. Content-type negotiation.

#### Technical Details

```
GET    /v2/{name}/manifests/{reference}   — manifest al (tag veya digest)
PUT    /v2/{name}/manifests/{reference}   — manifest yükle
DELETE /v2/{name}/manifests/{reference}   — manifest sil
HEAD   /v2/{name}/manifests/{reference}   — manifest varlık kontrolü
```

Content-Types:
- `application/vnd.docker.distribution.manifest.v2+json`
- `application/vnd.oci.image.manifest.v1+json`
- `application/vnd.docker.distribution.manifest.list.v2+json` (multi-arch)

#### Files to Touch

- `internal/plugins/docker/manifest.go` (new)
- `internal/plugins/docker/manifest_test.go` (new)

#### Dependencies

- T054, T027 (storage)

#### Success Criteria

- [ ] Manifest GET by tag çalışıyor
- [ ] Manifest GET by digest çalışıyor
- [ ] Manifest PUT çalışıyor (push)
- [ ] Content-type negotiation doğru
- [ ] Multi-arch manifest list desteği

---

### T056: Blob Operations

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

Image layer blob upload/download. Chunked upload desteği, digest verification.

#### Technical Details

```
HEAD   /v2/{name}/blobs/{digest}                  — blob varlık kontrolü
GET    /v2/{name}/blobs/{digest}                  — blob indir
POST   /v2/{name}/blobs/uploads/                  — upload başlat
PATCH  /v2/{name}/blobs/uploads/{uuid}            — chunk yükle
PUT    /v2/{name}/blobs/uploads/{uuid}?digest=... — upload tamamla
DELETE /v2/{name}/blobs/{digest}                  — blob sil
```

#### Files to Touch

- `internal/plugins/docker/blob.go` (new)
- `internal/plugins/docker/upload.go` (new)
- `internal/plugins/docker/blob_test.go` (new)

#### Dependencies

- T054, T027 (storage)

#### Success Criteria

- [ ] Blob download çalışıyor
- [ ] Monolithic upload çalışıyor
- [ ] Chunked upload çalışıyor
- [ ] Digest verification yapılıyor
- [ ] Cross-repository blob mounting

---

### T057: Docker Hub Upstream Sync

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Docker Hub'dan image metadata çekme, manifest ve layer'ları indirme. Docker Hub auth (token based), rate limit handling.

#### Files to Touch

- `internal/plugins/docker/upstream.go` (new)
- `internal/plugins/docker/upstream_test.go` (new)

#### Dependencies

- T054, T040 (sync engine)

#### Success Criteria

- [ ] Docker Hub'dan metadata çekiliyor
- [ ] Image pull (manifest + layers) çalışıyor
- [ ] Auth token flow çalışıyor
- [ ] Rate limit'e takılınca retry with backoff

---

### T058: Docker Plugin Config ve Entegrasyon

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

Docker plugin'in RegistryPlugin interface'ini tam implement etmesi, config, default values.

#### Files to Touch

- `internal/plugins/docker/config.go` (new)
- `internal/plugins/docker/plugin.go` (update)

#### Dependencies

- T054-T057

#### Success Criteria

- [ ] RegistryPlugin interface tam implement edildi
- [ ] DefaultConfig() anlamlı varsayılanlar döndürüyor
- [ ] Configure() config'i doğruluyor
- [ ] End-to-end: config → register → serve → pull/push

## Performance Targets

- Image pull (cache hit): < 100ms per layer
- Blob upload throughput: limited by network/disk I/O
- Manifest operations: < 10ms

## Risk Assessment

- **Yüksek Risk:** Docker Registry API v2 spesifikasyonu karmaşık (chunked upload, content negotiation)
- **Çözüm:** OCI distribution spec referans alınacak, Docker client ile entegrasyon testi zorunlu
- **Dikkat:** Docker Hub rate limiting (unauthenticated: 100 pulls/6h)
