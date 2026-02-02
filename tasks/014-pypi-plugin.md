# Feature 014: PyPI Registry Plugin

**Feature ID:** F014
**Feature Name:** PyPI Registry Plugin
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

PEP 503 Simple Repository API uyumlu PyPI plugin. pip install, twine upload desteği. pypi.org upstream sync. Wheel ve sdist formatları.

## Goals

- PEP 503 Simple API tam uyumluluk
- `pip install pkg` ile paket indirilebilmeli
- `twine upload` ile paket yüklenebilmeli
- pypi.org'dan sync

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T064-T067)
- [ ] `pip install` çalışıyor
- [ ] `twine upload` çalışıyor
- [ ] Upstream sync çalışıyor

## Tasks

### T064: PEP 503 Simple API

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

PyPI Simple Repository API endpoint'leri. Paket listesi ve versiyon indeks sayfaları (HTML).

#### Technical Details

```
GET /pypi/simple/                     — tüm paketlerin HTML indeksi
GET /pypi/simple/{package}/           — paketin tüm dosyalarının HTML indeksi
```

PEP 503 formatı: HTML sayfaları anchor tag'leri ile dosya linkleri.
PEP 691: JSON response desteği (Content-Type: application/vnd.pypi.simple.v1+json)

#### Files to Touch

- `internal/plugins/pypi/plugin.go` (new)
- `internal/plugins/pypi/routes.go` (new)
- `internal/plugins/pypi/simple.go` (new)

#### Dependencies

- T033, T019

#### Success Criteria

- [ ] Simple index HTML formatında çalışıyor
- [ ] Paket indeks sayfası dosya hash'lerini içeriyor
- [ ] PEP 691 JSON response destekleniyor
- [ ] Package name normalizasyonu doğru (PEP 503)

---

### T065: Package File Serve ve Upload

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Wheel ve sdist dosyalarını serve etme. Twine upload (multipart form) desteği.

#### Technical Details

```
GET  /pypi/packages/{path}           — dosya indir
POST /pypi/                          — upload (multipart/form-data, twine uyumlu)
```

Upload fields: `:action` = `file_upload`, `content`, `name`, `version`, `filetype`, `metadata_version`

#### Files to Touch

- `internal/plugins/pypi/serve.go` (new)
- `internal/plugins/pypi/upload.go` (new)
- `internal/plugins/pypi/upload_test.go` (new)

#### Dependencies

- T064, T027

#### Success Criteria

- [ ] Wheel/sdist download çalışıyor
- [ ] Hash (SHA256) doğrulaması yapılıyor
- [ ] Twine upload çalışıyor
- [ ] Duplicate version upload engelleneiyor

---

### T066: pypi.org Upstream Sync

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

pypi.org'dan metadata ve dosya çekme. JSON API ve Simple API kullanarak sync.

#### Files to Touch

- `internal/plugins/pypi/upstream.go` (new)

#### Dependencies

- T064, T040

#### Success Criteria

- [ ] Upstream'den metadata çekiliyor
- [ ] Wheel/sdist dosyaları indiriliyor
- [ ] Hash doğrulaması yapılıyor
- [ ] Yanked versiyon bilgisi korunuyor

---

### T067: PyPI Plugin Config ve Entegrasyon

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

PyPI plugin tam entegrasyonu, RegistryPlugin interface implementasyonu.

#### Files to Touch

- `internal/plugins/pypi/config.go` (new)
- `internal/plugins/pypi/plugin.go` (update)

#### Dependencies

- T064-T066

#### Success Criteria

- [ ] RegistryPlugin interface tam implement edildi
- [ ] End-to-end pip install ve twine upload çalışıyor

## Risk Assessment

- **Düşük Risk:** PEP 503 basit bir API (HTML sayfaları)
- **Dikkat:** PyPI metadata formatı çeşitli versiyonlarda farklılık gösterebilir
