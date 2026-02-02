# Feature 020: CLI Tool (kantarctl)

**Feature ID:** F020
**Feature Name:** CLI Tool (kantarctl)
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 2 weeks
**Status:** NOT_STARTED

## Overview

cobra tabanlı CLI aracı. Registry yönetimi, paket onaylama/engelleme, kullanıcı yönetimi, bulk import/export, sistem bakımı (GC, backup). lipgloss ile stil. REST API client olarak çalışır.

## Goals

- PRD Section 5.5'teki tüm CLI komutlarını implemente etme
- Tutarlı ve kullanıcı dostu CLI deneyimi
- Renkli ve formatlanmış çıktı
- Shell completion desteği

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T086-T092)
- [ ] Tüm alt komutlar çalışıyor
- [ ] Çıktı formatı tutarlı ve okunabilir
- [ ] Help metinleri yeterli

## Tasks

### T086: CLI Framework ve API Client

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

cobra ile CLI framework kurulumu, Kantar API'sine bağlanan HTTP client, kimlik doğrulama (token), output formatting (JSON/table/text).

#### Technical Details

```go
kantarctl --server http://kantar.local:8080 --token xxx

// veya config dosyası:
// ~/.kantarctl.toml
// [server]
// url = "http://kantar.local:8080"
// token = "xxx"
```

#### Files to Touch

- `cmd/kantarctl/main.go` (update)
- `cmd/kantarctl/root.go` (new)
- `internal/cli/client.go` (new)
- `internal/cli/output.go` (new)
- `internal/cli/config.go` (new)

#### Dependencies

- T001, T005 (cobra bağımlılığı)

#### Success Criteria

- [ ] Root command ve help çalışıyor
- [ ] API client bağlantı kurabiliyor
- [ ] Token auth çalışıyor
- [ ] JSON ve table output formatları destekleniyor

---

### T087: Registry Komutları

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Registry yönetim komutları: list, add, sync, status.

#### Technical Details

```bash
kantarctl registry list
kantarctl registry add npm --upstream https://registry.npmjs.org
kantarctl registry sync npm
kantarctl registry sync npm --package express
```

#### Files to Touch

- `cmd/kantarctl/registry.go` (new)

#### Dependencies

- T086

#### Success Criteria

- [ ] `registry list` çalışıyor
- [ ] `registry sync` çalışıyor (tekil ve toplu)
- [ ] Çıktı formatı düzgün (tablo)

---

### T088: Package Komutları

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Paket yönetim komutları: search, approve, block, info, deps.

#### Technical Details

```bash
kantarctl package search express --registry npm
kantarctl package approve express@4.21.2 --registry npm
kantarctl package approve "express@4.*" --registry npm
kantarctl package block malicious-pkg --registry npm --reason "supply-chain risk"
kantarctl package info express --registry npm
kantarctl package deps express@4.21.2 --registry npm --tree
```

#### Files to Touch

- `cmd/kantarctl/package.go` (new)

#### Dependencies

- T086

#### Success Criteria

- [ ] Search sonuçları düzgün gösteriliyor
- [ ] Approve/block komutları çalışıyor
- [ ] Glob pattern desteği çalışıyor
- [ ] Dependency tree formatında çıktı

---

### T089: Bulk Import/Export Komutları

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Toplu paket onaylama (TOML dosyasından import) ve mevcut listeyi export etme.

```bash
kantarctl package import --file approved-packages.toml
kantarctl package export --registry npm --format toml
```

#### Files to Touch

- `cmd/kantarctl/import_export.go` (new)

#### Dependencies

- T086

#### Success Criteria

- [ ] TOML import çalışıyor
- [ ] TOML/JSON export çalışıyor
- [ ] İlerleme gösterimi (progress bar)

---

### T090: User Komutları

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Kullanıcı yönetim komutları: list, create, token oluşturma.

```bash
kantarctl user list
kantarctl user create --username ali --role consumer
kantarctl user token create --username ci-runner --expires 90d
```

#### Files to Touch

- `cmd/kantarctl/user.go` (new)

#### Dependencies

- T086

#### Success Criteria

- [ ] Kullanıcı listeleme ve oluşturma çalışıyor
- [ ] Token oluşturma çalışıyor
- [ ] Rol ataması çalışıyor

---

### T091: Policy Komutları

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

Policy yönetim komutları: validate, test.

```bash
kantarctl policy validate
kantarctl policy test express@4.21.2 --registry npm
```

#### Files to Touch

- `cmd/kantarctl/policy.go` (new)

#### Dependencies

- T086

#### Success Criteria

- [ ] Policy validate çalışıyor
- [ ] Policy test çalışıyor ve sonucu gösteriyor

---

### T092: System Komutları

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Sistem bakım komutları: status, gc, backup, restore, serve, init, install-service.

```bash
kantarctl status
kantarctl gc
kantarctl backup --output /backup/kantar.tar
kantarctl restore --input /backup/kantar.tar
```

#### Files to Touch

- `cmd/kantarctl/system.go` (new)
- `cmd/kantar/serve.go` (new)
- `cmd/kantar/init.go` (new)

#### Dependencies

- T086

#### Success Criteria

- [ ] Status komutu tüm registry'lerin durumunu gösteriyor
- [ ] GC komutu çalışıyor ve rapor döndürüyor
- [ ] Backup/restore çalışıyor
- [ ] `kantar serve` komutu server başlatıyor
- [ ] `kantar init` komutu varsayılan config oluşturuyor

## Risk Assessment

- **Düşük Risk:** cobra çok olgun bir CLI framework
- **Dikkat:** CLI ve API arasındaki tutarlılık — her CLI komutu bir API endpoint'ini çağırmalı
