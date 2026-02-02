# Feature 003: Database Layer

**Feature ID:** F003
**Feature Name:** Database Layer
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1.5 weeks
**Status:** NOT_STARTED

## Overview

SQLite (varsayılan) ve PostgreSQL destekli veritabanı katmanı. sqlc ile type-safe SQL generation, migration sistemi, connection pooling. Tüm veri modelleri (paketler, kullanıcılar, roller, audit, policy) için şema tanımları.

## Goals

- Dual database desteği (SQLite + PostgreSQL)
- sqlc ile type-safe query'ler
- Migration sistemi ile şema versiyonlama
- Connection pooling ve sağlık kontrolü

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T010-T014)
- [ ] SQLite ile geliştirme ortamında çalışıyor
- [ ] PostgreSQL ile production ortamında çalışıyor
- [ ] Migration'lar idempotent çalışıyor
- [ ] sqlc generate hatasız çalışıyor

## Tasks

### T010: Database Interface ve Factory

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Veritabanı soyutlama katmanı. Config'e göre SQLite veya PostgreSQL bağlantısı oluşturan factory pattern.

#### Technical Details

```go
type DB interface {
    Queries() *sqlc.Queries
    Close() error
    Ping(ctx context.Context) error
    Migrate(ctx context.Context) error
}
```

#### Files to Touch

- `internal/database/database.go` (new)
- `internal/database/sqlite.go` (new)
- `internal/database/postgres.go` (new)
- `internal/database/factory.go` (new)

#### Dependencies

- T005 (SQLite/PG driver bağımlılıkları)
- T006 (config struct'ları)

#### Success Criteria

- [ ] Interface tanımlı
- [ ] SQLite implementasyonu çalışıyor
- [ ] PostgreSQL implementasyonu çalışıyor
- [ ] Config'e göre doğru backend seçiliyor

---

### T011: SQL Şema Tanımları

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

Tüm veritabanı tablolarının şema tanımları. Paketler, versiyonlar, kullanıcılar, roller, token'lar, audit, policy kayıtları.

#### Technical Details

Ana tablolar:
- `users` — kullanıcı hesapları
- `roles` — RBAC rolleri
- `user_roles` — kullanıcı-rol ilişkisi
- `api_tokens` — API token'ları
- `registries` — registry yapılandırmaları
- `packages` — paket metadata
- `package_versions` — paket versiyonları
- `package_approvals` — onay/engelleme kayıtları
- `policies` — policy tanımları
- `audit_logs` — audit kayıtları
- `sync_jobs` — senkronizasyon görevleri

#### Files to Touch

- `migrations/001_initial_schema.sql` (new)
- `internal/database/schema/` (new — sqlc schema dosyaları)

#### Dependencies

- T010

#### Success Criteria

- [ ] Tüm tablolar tanımlı
- [ ] İndeksler uygun alanlarda
- [ ] Foreign key ilişkileri doğru
- [ ] SQLite ve PostgreSQL uyumlu SQL

---

### T012: Migration Sistemi

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Embedded SQL migration dosyaları ile otomatik şema migration'ı. Go `embed` ile migration dosyaları binary'ye gömülecek.

#### Technical Details

- `golang-migrate` veya basit custom migration runner
- Migration dosyaları `migrations/` altında numaralı
- Uygulama başlangıcında otomatik migration
- Rollback desteği (down migration'lar)

#### Files to Touch

- `internal/database/migrate.go` (new)
- `internal/database/migrate_test.go` (new)
- `migrations/` (embed directive)

#### Dependencies

- T011

#### Success Criteria

- [ ] Migration'lar otomatik uygulanıyor
- [ ] Aynı migration iki kez uygulanmıyor (idempotent)
- [ ] Migration sürüm takibi çalışıyor
- [ ] Boş DB'den tam şema oluşturuluyor

---

### T013: sqlc Query Tanımları

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

sqlc ile type-safe SQL query'lerinin tanımlanması. CRUD operasyonları, filtreleme, pagination.

#### Technical Details

```sql
-- name: GetPackage :one
SELECT * FROM packages WHERE registry_type = ? AND name = ?;

-- name: ListPackages :many
SELECT * FROM packages WHERE registry_type = ? ORDER BY name LIMIT ? OFFSET ?;

-- name: CreatePackage :exec
INSERT INTO packages (registry_type, name, ...) VALUES (?, ?, ...);
```

#### Files to Touch

- `sqlc.yaml` (new)
- `internal/database/queries/*.sql` (new)
- `internal/database/sqlc/` (generated)

#### Dependencies

- T011

#### Success Criteria

- [ ] `sqlc generate` hatasız çalışıyor
- [ ] Tüm CRUD operasyonları için query'ler tanımlı
- [ ] Üretilen Go kodu derleniyor
- [ ] Pagination ve filtreleme query'leri var

---

### T014: Database Test Altyapısı

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

In-memory SQLite ile test helper'ları. Her test için temiz database, test fixture'ları.

#### Files to Touch

- `internal/database/testutil.go` (new)
- `internal/database/database_test.go` (new)

#### Dependencies

- T010, T012, T013

#### Success Criteria

- [ ] Test helper ile temiz DB oluşturulabiliyor
- [ ] Migration'lar test DB'de çalışıyor
- [ ] CRUD operasyonları test edildi
- [ ] Test'ler paralel çalışabiliyor (izole DB'ler)

## Performance Targets

- Query response time: < 5ms (SQLite), < 10ms (PostgreSQL remote)
- Connection pool size: configurable, default 10

## Risk Assessment

- **Orta Risk:** Dual database desteği (SQLite/PG) SQL dialect farklılıkları
- **Çözüm:** sqlc her iki dialect için kod üretebilir; migration SQL'leri ortak tutulmalı
