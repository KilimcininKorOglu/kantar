# Feature 016: Cargo Registry Plugin

**Feature ID:** F016
**Feature Name:** Cargo Registry Plugin
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Cargo Sparse Registry (RFC 2789) uyumlu Rust crate registry. `cargo install`, `cargo publish` desteği. crates.io upstream sync.

## Goals

- Sparse Registry protocol uyumluluk
- `cargo install` ve `cargo publish` uyumluluğu
- crates.io'dan sync

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T072-T075)
- [ ] `cargo install` çalışıyor
- [ ] `cargo publish` çalışıyor
- [ ] Upstream sync çalışıyor

## Tasks

### T072: Cargo Sparse Registry Protocol

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Sparse Registry endpoint'leri: config.json, crate metadata, API.

#### Technical Details

```
GET  /cargo/config.json                 — registry config
GET  /cargo/{prefix}/{crate}            — crate metadata (index entry)
GET  /cargo/api/v1/crates               — crate arama
POST /cargo/api/v1/crates/new           — publish
GET  /cargo/api/v1/crates/{crate}/{version}/download — crate indir
```

Prefix: 1-char → `1/`, 2-char → `2/`, 3-char → `3/{first-char}/`, 4+ → `{first-two}/{second-two}/`

#### Files to Touch

- `internal/plugins/cargo/plugin.go` (new)
- `internal/plugins/cargo/routes.go` (new)
- `internal/plugins/cargo/index.go` (new)

#### Dependencies

- T033, T019

#### Success Criteria

- [ ] config.json endpoint çalışıyor
- [ ] Crate index entry doğru formatta
- [ ] Prefix hesaplama doğru
- [ ] Search endpoint çalışıyor

---

### T073: Crate Download ve Publish

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Crate dosyası download ve publish. Publish auth, metadata kayıt.

#### Files to Touch

- `internal/plugins/cargo/download.go` (new)
- `internal/plugins/cargo/publish.go` (new)

#### Dependencies

- T072, T027, T022

#### Success Criteria

- [ ] Crate download çalışıyor
- [ ] `cargo publish` çalışıyor
- [ ] Auth token doğrulanıyor
- [ ] Checksum doğrulaması yapılıyor

---

### T074: crates.io Upstream Sync

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

crates.io'dan crate metadata ve dosya çekme.

#### Files to Touch

- `internal/plugins/cargo/upstream.go` (new)

#### Dependencies

- T072, T040

#### Success Criteria

- [ ] Upstream'den metadata çekiliyor
- [ ] Crate dosyaları indiriliyor
- [ ] Yanked bilgisi korunuyor

---

### T075: Cargo Plugin Config ve Entegrasyon

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

Cargo plugin tam entegrasyonu.

#### Files to Touch

- `internal/plugins/cargo/config.go` (new)
- `internal/plugins/cargo/plugin.go` (update)

#### Dependencies

- T072-T074

#### Success Criteria

- [ ] RegistryPlugin interface tam implement edildi
- [ ] End-to-end cargo install/publish çalışıyor

## Risk Assessment

- **Orta Risk:** Sparse Registry protocol nispeten yeni (RFC 2789)
- **Çözüm:** crates.io implementasyonunu referans alma
