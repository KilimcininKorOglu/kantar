# Feature 002: Configuration System

**Feature ID:** F002
**Feature Name:** Configuration System
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

TOML tabanlı yapılandırma sistemi. Ana config dosyası (`kantar.toml`), policy dosyaları, environment variable interpolasyonu, config doğrulama ve varsayılan değerler. `kantar init` komutu ile varsayılan config oluşturma.

## Goals

- TOML tabanlı yapılandırma okuma/yazma
- Environment variable interpolasyonu (`${VAR}` syntax)
- Config doğrulama (geçersiz değerler için anlaşılır hatalar)
- Varsayılan config üretimi (`kantar init`)

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T006-T009)
- [ ] `kantar.toml` parse ediliyor
- [ ] Env var interpolasyonu çalışıyor
- [ ] Geçersiz config'de anlaşılır hata mesajı veriliyor
- [ ] `kantar init` varsayılan config dosyası üretiyor

## Tasks

### T006: Config Struct Tanımları

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Go struct'ları ile config yapısını modelleme. Server, Storage, Database, Auth, Cache, Logging, Notifications bölümleri.

#### Technical Details

```go
type Config struct {
    Server        ServerConfig
    Storage       StorageConfig
    Database      DatabaseConfig
    Auth          AuthConfig
    Cache         CacheConfig
    Logging       LoggingConfig
    Notifications NotificationsConfig
    Registries    map[string]RegistryConfig
}
```

PRD Section 7.1'deki tüm alanları kapsamalı.

#### Files to Touch

- `internal/config/config.go` (new)
- `internal/config/defaults.go` (new)

#### Dependencies

- T001 (proje yapısı)

#### Success Criteria

- [ ] Tüm config alanları struct'larda tanımlı
- [ ] TOML tag'leri doğru
- [ ] Varsayılan değerler tanımlı
- [ ] Unit test'ler yazıldı

---

### T007: TOML Parser ve Env Var Interpolasyonu

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

TOML dosyasını okuma, `${ENV_VAR}` syntax'ını environment variable değerleri ile değiştirme, config struct'a deserialize etme.

#### Technical Details

- BurntSushi/toml kütüphanesi kullanılacak
- `${VAR}` ve `${VAR:-default}` syntax'ı desteklenecek
- Config dosyası yolu: CLI flag, env var (`KANTAR_CONFIG`), varsayılan (`/etc/kantar/kantar.toml`, `./kantar.toml`)

#### Files to Touch

- `internal/config/loader.go` (new)
- `internal/config/envvar.go` (new)
- `internal/config/loader_test.go` (new)

#### Dependencies

- T006

#### Success Criteria

- [ ] TOML dosyası okunuyor ve struct'a deserialize ediliyor
- [ ] `${VAR}` interpolasyonu çalışıyor
- [ ] `${VAR:-default}` fallback çalışıyor
- [ ] Dosya bulunamadığında anlaşılır hata
- [ ] Unit test'ler yazıldı

---

### T008: Config Doğrulama

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Config değerlerini doğrulama: geçerli port aralığı, var olan dizinler, geçerli URL'ler, tutarlı ayarlar (örn: storage type "s3" ise s3 config zorunlu).

#### Files to Touch

- `internal/config/validate.go` (new)
- `internal/config/validate_test.go` (new)

#### Dependencies

- T006

#### Success Criteria

- [ ] Geçersiz port değerinde hata
- [ ] Eksik zorunlu alanlar tespit ediliyor
- [ ] Tutarsız config kombinasyonları uyarılıyor
- [ ] Hata mesajları alan adı ve beklenen değeri içeriyor

---

### T009: Config Init Komutu

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

`kantar init` komutu ile varsayılan `kantar.toml` ve `policies/security.toml` dosyalarını oluşturma.

#### Files to Touch

- `internal/config/init.go` (new)
- `internal/config/templates.go` (new — embedded template dosyaları)

#### Dependencies

- T006, T007

#### Success Criteria

- [ ] `kantar init` çalıştırıldığında varsayılan config dosyaları oluşuyor
- [ ] Mevcut dosyaların üzerine yazılmadan önce onay isteniyor
- [ ] Oluşturulan config geçerli ve parse edilebilir

## Risk Assessment

- **Düşük Risk:** TOML parsing iyi bilinen bir problem
- **Dikkat:** Env var interpolasyonu güvenlik açısından dikkatli yapılmalı (secret leak önleme)
