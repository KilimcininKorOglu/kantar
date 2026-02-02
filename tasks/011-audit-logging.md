# Feature 011: Audit Logging System

**Feature ID:** F011
**Feature Name:** Audit Logging System
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Tüm sistem olaylarının yapılandırılmış JSON formatında kaydedildiği audit log sistemi. Hash-chain ile tamper detection (blockchain tarzı). Olay türleri: package.download, package.approve, user.login, policy.violation, vb. Filtreleme ve export (CSV/JSON) desteği.

## Goals

- Yapılandırılmış JSON audit log
- Hash-chain ile değiştirilmezlik garantisi
- Filtreleme ve arama
- CSV/JSON export

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T049-T053)
- [ ] Tüm olay türleri loglanıyor
- [ ] Hash-chain doğrulaması çalışıyor
- [ ] Filtreleme ve export çalışıyor

## Tasks

### T049: Audit Logger Core

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Audit log yapısı, event oluşturma, dosyaya yazma. PRD Section 5.7'deki JSON formatına uygun.

#### Technical Details

```go
type AuditEvent struct {
    Timestamp  time.Time     `json:"timestamp"`
    Event      string        `json:"event"`
    Actor      Actor         `json:"actor"`
    Resource   Resource      `json:"resource"`
    Result     string        `json:"result"`
    Metadata   EventMetadata `json:"metadata"`
    PrevHash   string        `json:"prev_hash"`
    Hash       string        `json:"hash"`
}

type AuditLogger interface {
    Log(ctx context.Context, event *AuditEvent) error
    Query(ctx context.Context, filter AuditFilter) ([]AuditEvent, error)
    Verify(ctx context.Context) (*VerifyResult, error)
}
```

#### Files to Touch

- `internal/audit/audit.go` (new)
- `internal/audit/types.go` (new)
- `internal/audit/logger.go` (new)

#### Dependencies

- T001

#### Success Criteria

- [ ] Audit event oluşturma ve loglanma çalışıyor
- [ ] JSON formatı PRD ile uyumlu
- [ ] Dosyaya ve DB'ye yazma destekleniyor
- [ ] Thread-safe yazma

---

### T050: Hash-Chain Tamper Detection

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Her audit log entry'si önceki entry'nin hash'ini içerir. SHA-256 ile hash chain oluşturma. Doğrulama fonksiyonu.

#### Technical Details

```
Entry N:
  prev_hash = SHA256(Entry N-1)
  hash = SHA256(timestamp + event + actor + resource + result + prev_hash)

Verification:
  1. İlk entry'den son entry'ye kadar hash zincirini doğrula
  2. Herhangi bir kopma → tamper detected
```

#### Files to Touch

- `internal/audit/hashchain.go` (new)
- `internal/audit/hashchain_test.go` (new)

#### Dependencies

- T049

#### Success Criteria

- [ ] Hash chain doğru oluşturuluyor
- [ ] Verify fonksiyonu tüm chain'i kontrol ediyor
- [ ] Değiştirilmiş entry tespit ediliyor
- [ ] Silinmiş entry tespit ediliyor
- [ ] Unit test'ler yazıldı

---

### T051: Audit Event Türleri ve Hook'lar

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

PRD'deki tüm olay türlerinin tanımlanması ve ilgili code path'lerine hook eklenmesi.

#### Technical Details

Event türleri:
- `package.download` / `package.upload` / `package.delete`
- `package.approve` / `package.block`
- `policy.violation` / `policy.update`
- `user.login` / `user.create` / `user.token.create`
- `registry.sync` / `registry.config.update`
- `system.gc` / `system.backup`

#### Files to Touch

- `internal/audit/events.go` (new)
- `internal/audit/middleware.go` (new — HTTP middleware ile otomatik loglama)

#### Dependencies

- T049, T016 (middleware zinciri)

#### Success Criteria

- [ ] Tüm olay türleri tanımlı
- [ ] HTTP istekleri middleware ile otomatik loglanıyor
- [ ] Actor bilgisi auth context'ten alınıyor
- [ ] İşlem sonucu (success/failure) doğru raporlanıyor

---

### T052: Audit Query ve Filtreleme

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Audit log'larda filtreleme ve arama. Tarih aralığı, actor, event türü, registry, paket bazlı filtreleme. Pagination desteği.

#### Files to Touch

- `internal/audit/query.go` (new)
- `internal/audit/query_test.go` (new)

#### Dependencies

- T049, T013 (DB query'leri — audit_logs tablosu)

#### Success Criteria

- [ ] Tarih aralığı filtreleme çalışıyor
- [ ] Actor bazlı filtreleme çalışıyor
- [ ] Event türü filtreleme çalışıyor
- [ ] Pagination doğru çalışıyor

---

### T053: Audit API ve Export

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Audit REST API endpoint'leri ve CSV/JSON export.

#### Technical Details

```
GET  /api/v1/audit                — filtrelenmiş audit listesi
GET  /api/v1/audit/export         — CSV/JSON export (?format=csv|json)
POST /api/v1/audit/verify         — hash chain doğrulama
```

#### Files to Touch

- `internal/server/handlers/audit.go` (new)
- `internal/audit/export.go` (new)

#### Dependencies

- T052, T019

#### Success Criteria

- [ ] Audit API endpoint'leri çalışıyor
- [ ] CSV export çalışıyor
- [ ] JSON export çalışıyor
- [ ] Verify endpoint hash chain sonucu dönüyor

## Performance Targets

- Audit write latency: < 2ms (async write)
- Query response: < 50ms (indexed queries)
- Export: streaming response (büyük veri setleri için)

## Risk Assessment

- **Düşük Risk:** JSON logging ve hash chain iyi bilinen yapılar
- **Dikkat:** Yüksek trafik altında audit yazma performansı — async/buffered writing önerilir
- **Dikkat:** Audit log dosya boyutu — log rotation gerekli
