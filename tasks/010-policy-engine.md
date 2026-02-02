# Feature 010: Policy Engine

**Feature ID:** F010
**Feature Name:** Policy Engine
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1.5 weeks
**Status:** NOT_STARTED

## Overview

Deklaratif TOML tabanlı policy engine. Lisans kontrolü, güvenlik açığı seviyesi, paket yaşı, boyut limiti, isimlendirme kuralları. Policy'ler paket onayında otomatik değerlendirilir. PRD Section 5.3'teki tüm policy tiplerini kapsar.

## Goals

- TOML tabanlı policy tanımlama
- Otomatik policy değerlendirme (approve/block/warn)
- Çoklu policy tipi desteği
- Policy test mekanizması

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T044-T048)
- [ ] Lisans policy çalışıyor
- [ ] Güvenlik policy çalışıyor
- [ ] Boyut ve yaş policy'leri çalışıyor
- [ ] Policy ihlali doğru action'ı tetikliyor (block/warn/log)

## Tasks

### T044: Policy Engine Core

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Policy engine'in çekirdek yapısı. Policy yükleme, değerlendirme pipeline'ı, sonuç birleştirme.

#### Technical Details

```go
type PolicyAction string
const (
    ActionBlock PolicyAction = "block"
    ActionWarn  PolicyAction = "warn"
    ActionLog   PolicyAction = "log"
    ActionAllow PolicyAction = "allow"
)

type PolicyResult struct {
    Allowed    bool
    Action     PolicyAction
    Violations []Violation
    Warnings   []Warning
}

type PolicyEngine struct {
    policies []Policy
}

func (pe *PolicyEngine) Evaluate(ctx, pkg *Package) (*PolicyResult, error)
```

#### Files to Touch

- `internal/policy/engine.go` (new)
- `internal/policy/types.go` (new)
- `internal/policy/engine_test.go` (new)

#### Dependencies

- T001

#### Success Criteria

- [ ] Policy pipeline çalışıyor
- [ ] Birden fazla policy sonucu birleştiriliyor (en kısıtlayıcı kazanır)
- [ ] Violation ve Warning detayları raporlanıyor
- [ ] Unit test'ler yazıldı

---

### T045: Policy TOML Loader

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

TOML formatındaki policy dosyalarını okuma ve Policy struct'larına deserialize etme. `policies/` dizinindeki tüm dosyaları tarama.

#### Files to Touch

- `internal/policy/loader.go` (new)
- `internal/policy/loader_test.go` (new)

#### Dependencies

- T044

#### Success Criteria

- [ ] TOML policy dosyaları okunuyor
- [ ] Birden fazla policy dosyası destekleniyor
- [ ] Geçersiz policy'de anlaşılır hata mesajı
- [ ] Policy hot-reload desteklenebilir yapıda

---

### T046: License Policy

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Lisans kontrolü policy'si. Allowed/blocked lisans listeleri, SPDX identifier'ları ile eşleştirme.

#### Technical Details

```toml
[policy.license]
allowed = ["MIT", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "ISC"]
blocked = ["GPL-3.0", "AGPL-3.0"]
action = "block"  # block | warn | log
```

#### Files to Touch

- `internal/policy/license.go` (new)
- `internal/policy/license_test.go` (new)

#### Dependencies

- T044

#### Success Criteria

- [ ] Allowed listesindeki lisanslar geçiyor
- [ ] Blocked listesindeki lisanslar engelleniyor
- [ ] Bilinmeyen lisanslarda configurable action
- [ ] SPDX identifier normalizasyonu

---

### T047: Size ve Age Policy

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Paket boyut limiti ve yaş kontrolü. Typosquatting koruması için yeni paketleri bekletme.

#### Technical Details

```toml
[policy.age]
min_package_age = "7d"
min_maintainers = 2

[policy.size]
max_package_size = "500MB"
max_layer_count = 20  # Docker specific
```

#### Files to Touch

- `internal/policy/size.go` (new)
- `internal/policy/age.go` (new)
- `internal/policy/size_test.go` (new)
- `internal/policy/age_test.go` (new)

#### Dependencies

- T044

#### Success Criteria

- [ ] Boyut limiti aşıldığında policy action tetikleniyor
- [ ] Min yaş kontrolü çalışıyor
- [ ] Min maintainer kontrolü çalışıyor
- [ ] Docker layer count kontrolü çalışıyor

---

### T048: Naming Policy ve Policy API

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1.5 days

#### Description

İsimlendirme kuralları (blocked scopes, prefixes) ve Policy management REST API. Policy doğrulama endpoint'i.

#### Technical Details

```toml
[policy.naming]
blocked_scopes = ["@evil-corp"]
blocked_prefixes = ["__test"]

[policy.version]
pin_strategy = "minor"
allow_prerelease = false
allow_deprecated = false
```

API:
```
GET  /api/v1/policies           — policy listele
PUT  /api/v1/policies/{name}    — policy güncelle
POST /api/v1/policies/validate  — paketi policy'ye karşı test et
```

#### Files to Touch

- `internal/policy/naming.go` (new)
- `internal/policy/version.go` (new)
- `internal/server/handlers/policies.go` (new)

#### Dependencies

- T044, T045, T019

#### Success Criteria

- [ ] Blocked scope/prefix kontrolü çalışıyor
- [ ] Version pin strategy çalışıyor
- [ ] Policy API endpoint'leri çalışıyor
- [ ] Policy validate endpoint bir paketi test edebiliyor

## Risk Assessment

- **Orta Risk:** Lisans tanımlama her ekosistemde farklı (npm: package.json, PyPI: metadata)
- **Çözüm:** Plugin FetchMetadata'da lisans bilgisini normalize ederek döndürecek
