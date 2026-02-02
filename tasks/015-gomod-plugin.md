# Feature 015: Go Modules Plugin

**Feature ID:** F015
**Feature Name:** Go Modules Plugin
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

GOPROXY protocol uyumlu Go module proxy. `go get`, `go mod download` komutlarıyla uyumlu. proxy.golang.org upstream desteği. Module zip ve go.mod dosyaları serve etme.

## Goals

- GOPROXY protocol tam uyumluluk
- `GOPROXY=http://kantar.local:8080/go go get pkg` çalışmalı
- proxy.golang.org'dan sync
- Private module desteği

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T068-T071)
- [ ] `go get` ile module indirilebiliyor
- [ ] Private module publish çalışıyor
- [ ] Upstream sync çalışıyor

## Tasks

### T068: GOPROXY Protocol Endpoint'leri

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Go Module Proxy protocol endpoint'leri: version list, info, mod, zip.

#### Technical Details

```
GET /go/{module}/@v/list             — versiyon listesi (text/plain, her satır bir versiyon)
GET /go/{module}/@v/{version}.info   — versiyon bilgisi (JSON)
GET /go/{module}/@v/{version}.mod    — go.mod dosyası
GET /go/{module}/@v/{version}.zip    — module zip
GET /go/{module}/@latest             — en son versiyon bilgisi
```

Module path encoding: büyük harfler `!` prefix ile escape edilir (örn: `github.com/!azure/...`)

#### Files to Touch

- `internal/plugins/gomod/plugin.go` (new)
- `internal/plugins/gomod/routes.go` (new)
- `internal/plugins/gomod/proxy.go` (new)

#### Dependencies

- T033, T019

#### Success Criteria

- [ ] Tüm GOPROXY endpoint'leri çalışıyor
- [ ] Module path encoding/decoding doğru
- [ ] Version list text/plain formatında
- [ ] .info JSON formatında (Version, Time alanları)

---

### T069: Module Storage ve Serve

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Module zip ve go.mod dosyalarını depolama ve serve etme. Checksum doğrulama.

#### Files to Touch

- `internal/plugins/gomod/storage.go` (new)
- `internal/plugins/gomod/serve.go` (new)

#### Dependencies

- T068, T027

#### Success Criteria

- [ ] Module zip doğru serve ediliyor
- [ ] go.mod dosyası doğru serve ediliyor
- [ ] Checksum (go.sum uyumlu) doğrulanıyor

---

### T070: proxy.golang.org Upstream Sync

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

proxy.golang.org'dan module çekme. sum.golang.org ile checksum doğrulama.

#### Files to Touch

- `internal/plugins/gomod/upstream.go` (new)

#### Dependencies

- T068, T040

#### Success Criteria

- [ ] Upstream'den module çekiliyor
- [ ] Checksum doğrulaması yapılıyor
- [ ] GONOSUMDB/GONOSUMCHECK private module'lar için çalışıyor

---

### T071: Go Modules Plugin Config ve Entegrasyon

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

Go Modules plugin tam entegrasyonu.

#### Files to Touch

- `internal/plugins/gomod/config.go` (new)
- `internal/plugins/gomod/plugin.go` (update)

#### Dependencies

- T068-T070

#### Success Criteria

- [ ] RegistryPlugin interface tam implement edildi
- [ ] End-to-end `go get` çalışıyor

## Risk Assessment

- **Düşük Risk:** GOPROXY protocol basit ve iyi belgelenmiş
- **Dikkat:** Module path encoding kuralları dikkat gerektirir
