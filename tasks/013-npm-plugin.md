# Feature 013: npm Registry Plugin

**Feature ID:** F013
**Feature Name:** npm Registry Plugin
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

npm Registry API uyumlu plugin. `npm install`, `npm publish`, `npm search` komutlarıyla uyumlu. npmjs.org upstream desteği. Scoped packages (@org/pkg) desteği.

## Goals

- npm CLI ile tam uyumluluk
- npmjs.org'dan allowlist paketleri sync
- Scoped package desteği
- Private package publishing

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T059-T063)
- [ ] `npm install pkg` ile paket indirilebiliyor
- [ ] `npm publish` ile paket yüklenebiliyor
- [ ] Upstream sync çalışıyor

## Tasks

### T059: npm Registry API — Package Metadata

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

npm paket metadata endpoint'leri: paket bilgisi, versiyon listesi, abbreviated metadata.

#### Technical Details

```
GET /npm/{package}                    — tam paket metadata (packument)
GET /npm/{package}/{version}          — tekil versiyon metadata
GET /npm/-/v1/search?text=            — arama
```

Packument format: CouchDB-style document with all versions, dist-tags, time info.

#### Files to Touch

- `internal/plugins/npm/plugin.go` (new)
- `internal/plugins/npm/routes.go` (new)
- `internal/plugins/npm/metadata.go` (new)

#### Dependencies

- T033 (RegistryPlugin interface), T019

#### Success Criteria

- [ ] Packument endpoint çalışıyor
- [ ] Scoped package'lar destekleniyor (@scope/pkg)
- [ ] Abbreviated metadata destekleniyor (Accept header)
- [ ] Search endpoint çalışıyor

---

### T060: npm Tarball Serve/Download

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

npm paket tarball'larını serve etme. SHA integrity check, content-length header.

#### Technical Details

```
GET /npm/{package}/-/{tarball}        — tarball indir
```

Tarball URL packument'taki `dist.tarball` alanında belirtilir.

#### Files to Touch

- `internal/plugins/npm/tarball.go` (new)
- `internal/plugins/npm/tarball_test.go` (new)

#### Dependencies

- T059, T027 (storage)

#### Success Criteria

- [ ] Tarball download çalışıyor
- [ ] SHA integrity doğrulanıyor
- [ ] Content-Length header doğru
- [ ] 404 olmayan tarball için uygun hata

---

### T061: npm Publish

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

npm publish desteği. Tarball upload, metadata kayıt, auth token doğrulama.

#### Technical Details

```
PUT /npm/{package}                    — publish (full packument with attachment)
DELETE /npm/{package}/-/{tarball}/-rev/{rev} — unpublish
POST /npm/-/user/org.couchdb.user:{user}    — login/create user
```

#### Files to Touch

- `internal/plugins/npm/publish.go` (new)
- `internal/plugins/npm/auth.go` (new)
- `internal/plugins/npm/publish_test.go` (new)

#### Dependencies

- T059, T060, T022 (auth middleware)

#### Success Criteria

- [ ] `npm publish` başarılı çalışıyor
- [ ] Auth token doğrulanıyor
- [ ] Duplicate version publish engelleneiyor
- [ ] Unpublish çalışıyor (yetki ile)

---

### T062: npmjs.org Upstream Sync

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

npmjs.org'dan paket metadata ve tarball çekme. Registry API uyumlu fetch.

#### Files to Touch

- `internal/plugins/npm/upstream.go` (new)
- `internal/plugins/npm/upstream_test.go` (new)

#### Dependencies

- T059, T040 (sync engine)

#### Success Criteria

- [ ] Upstream'den metadata çekiliyor
- [ ] Tarball indiriliyor ve doğrulanıyor
- [ ] Scoped package sync çalışıyor
- [ ] 404/429 hataları handle ediliyor

---

### T063: npm Plugin Config ve Entegrasyon

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

npm plugin tam entegrasyonu. RegistryPlugin interface implementasyonu, config, dist-tags yönetimi.

#### Files to Touch

- `internal/plugins/npm/config.go` (new)
- `internal/plugins/npm/plugin.go` (update)

#### Dependencies

- T059-T062

#### Success Criteria

- [ ] RegistryPlugin interface tam implement edildi
- [ ] End-to-end: npm install, npm publish çalışıyor
- [ ] dist-tags (latest, next) yönetimi çalışıyor

## Risk Assessment

- **Orta Risk:** npm packument formatı karmaşık (CouchDB document yapısı)
- **Çözüm:** npmjs.org response'larını referans alarak reverse-engineer
- **Dikkat:** npm CLI farklı versiyonları farklı endpoint'ler kullanabiliyor
