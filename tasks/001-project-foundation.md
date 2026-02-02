# Feature 001: Project Foundation

**Feature ID:** F001
**Feature Name:** Project Foundation & Build System
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Go projesinin temel yapısını oluşturur: dizin yapısı, Go modules, Makefile, CI/CD pipeline, linting ve temel bağımlılıklar. Tüm diğer feature'lar bu temele bağımlıdır.

## Goals

- Standart Go proje yapısını oluşturmak
- Tekrarlanabilir build sistemi kurmak
- CI/CD pipeline ile otomatik kalite kontrolü sağlamak
- Geliştirici deneyimini (DX) en baştan doğru kurmak

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T001-T005)
- [ ] `make build` ile tek binary üretilebiliyor
- [ ] `make test` ile testler çalışıyor
- [ ] `make lint` ile kod kalitesi kontrol ediliyor
- [ ] CI pipeline PR'larda otomatik çalışıyor

## Tasks

### T001: Go Module ve Dizin Yapısı

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Go module başlatma, standart Go proje yapısını oluşturma. cmd/, internal/, pkg/ dizinleri, ana entry point.

#### Technical Details

```
kantar/
  cmd/
    kantar/          # Ana binary entry point
      main.go
    kantarctl/       # CLI tool entry point
      main.go
  internal/
    config/          # Yapılandırma
    server/          # HTTP server
    auth/            # Kimlik doğrulama
    storage/         # Depolama katmanı
    cache/           # Önbellek katmanı
    plugin/          # Plugin sistemi
    policy/          # Policy engine
    audit/           # Audit logging
    database/        # Veritabanı katmanı
    model/           # Veri modelleri
  pkg/
    registry/        # RegistryPlugin interface (public API)
  web/               # React UI (ayrı build)
  migrations/        # SQL migration dosyaları
  scripts/           # Build/deploy scriptleri
  docs/              # Belgeler
```

#### Files to Touch

- `go.mod` (new)
- `go.sum` (new)
- `cmd/kantar/main.go` (new)
- `cmd/kantarctl/main.go` (new)
- `internal/` (new — dizin yapısı)
- `pkg/registry/plugin.go` (new)

#### Dependencies

- None (ilk task)

#### Success Criteria

- [ ] `go mod init` başarılı
- [ ] `go build ./...` hatasız
- [ ] Dizin yapısı oluşturuldu
- [ ] Entry point dosyaları derleniyor

---

### T002: Makefile ve Build Sistemi

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Makefile ile standart build, test, lint, clean komutlarını tanımlama. Versiyon bilgisi ldflags ile binary'ye gömülecek.

#### Technical Details

```makefile
# Hedefler: build, test, lint, clean, run, fmt, vet, generate
# ldflags: -X main.version, -X main.commit, -X main.buildDate
# Cross-compilation: GOOS/GOARCH desteği
```

#### Files to Touch

- `Makefile` (new)
- `cmd/kantar/main.go` (update — version vars)

#### Dependencies

- T001 (Go module mevcut olmalı)

#### Success Criteria

- [ ] `make build` tek binary üretiyor
- [ ] `make test` testleri çalıştırıyor
- [ ] `make lint` golangci-lint çalıştırıyor
- [ ] Binary versiyon bilgisi içeriyor (`kantar --version`)

---

### T003: Linting ve Kod Kalitesi Yapılandırması

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

golangci-lint yapılandırması, .editorconfig, .gitignore dosyaları.

#### Files to Touch

- `.golangci.yml` (new)
- `.editorconfig` (new)
- `.gitignore` (update)

#### Dependencies

- T001

#### Success Criteria

- [ ] `golangci-lint run` hatasız çalışıyor
- [ ] Tutarlı kod formatı sağlanıyor
- [ ] Gereksiz dosyalar gitignore'da

---

### T004: CI/CD Pipeline (GitHub Actions)

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

GitHub Actions ile build, test, lint pipeline'ı. PR'larda otomatik çalışacak. Release workflow ile binary dağıtımı.

#### Files to Touch

- `.github/workflows/ci.yml` (new)
- `.github/workflows/release.yml` (new)

#### Dependencies

- T002 (Makefile mevcut olmalı)

#### Success Criteria

- [ ] PR'larda CI otomatik çalışıyor
- [ ] Build, test, lint adımları başarılı
- [ ] Release tag'inde binary'ler otomatik yayınlanıyor

---

### T005: Temel Bağımlılıkların Eklenmesi

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 day

#### Description

Projenin ihtiyaç duyacağı temel Go bağımlılıklarını ekleme: chi, cobra, viper (TOML), sqlc, jwt-go, bcrypt.

#### Technical Details

```
github.com/go-chi/chi/v5           # HTTP router
github.com/spf13/cobra             # CLI framework
github.com/BurntSushi/toml         # TOML parser
github.com/golang-jwt/jwt/v5       # JWT
golang.org/x/crypto                # bcrypt
github.com/mattn/go-sqlite3        # SQLite driver
github.com/lib/pq                  # PostgreSQL driver
github.com/charmbracelet/lipgloss  # CLI styling
```

#### Files to Touch

- `go.mod` (update)
- `go.sum` (update)

#### Dependencies

- T001

#### Success Criteria

- [ ] Tüm bağımlılıklar `go.mod`'da
- [ ] `go mod tidy` temiz çalışıyor
- [ ] Import'lar derleniyor

## Risk Assessment

- **Düşük Risk:** Standart Go proje kurulumu, iyi bilinen araçlar
- **Dikkat:** Go versiyon uyumluluğu (1.23+ gerekli)

## Notes

- Web UI (React) build'i F021'de ele alınacak, burada sadece `web/` dizini oluşturulacak
- sqlc generate komutu F003'te yapılacak
