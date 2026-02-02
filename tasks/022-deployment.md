# Feature 022: Deployment & Distribution

**Feature ID:** F022
**Feature Name:** Deployment & Distribution
**Priority:** P2 - HIGH
**Target Version:** v1.0.0
**Estimated Duration:** 1.5 weeks
**Status:** NOT_STARTED

## Overview

Kantar'ın dağıtım ve deploy mekanizmaları: tek binary dağıtımı, Docker image, systemd service, Helm chart. Cross-platform build (Linux, macOS, Windows).

## Goals

- Tek binary ile kurulum (< 5 dakika)
- Docker Compose ile kolay deploy
- Kubernetes Helm chart
- Systemd service kurulumu

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T101-T106)
- [ ] Cross-platform binary'ler build ediliyor
- [ ] Docker image çalışıyor
- [ ] `kantar install-service` systemd unit kuruyor

## Tasks

### T101: Multi-Stage Dockerfile

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Multi-stage Docker build: Go build + React build → minimal runtime image.

#### Technical Details

```dockerfile
# Stage 1: React build
FROM node:22-alpine AS web-builder
# npm install && npm run build

# Stage 2: Go build
FROM golang:1.23-alpine AS go-builder
# COPY web build output
# go build with embed

# Stage 3: Runtime
FROM alpine:3.20
# COPY binary, minimal runtime
```

#### Files to Touch

- `Dockerfile` (new)
- `.dockerignore` (new)

#### Dependencies

- T001, T093

#### Success Criteria

- [ ] Docker image build ediliyor
- [ ] Image boyutu < 50MB
- [ ] Container başlıyor ve çalışıyor
- [ ] Health check çalışıyor

---

### T102: Docker Compose Dosyası

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 day

#### Description

PRD Section 8.2'deki Docker Compose yapılandırması: Kantar + opsiyonel PostgreSQL.

#### Files to Touch

- `docker-compose.yml` (new)
- `docker-compose.prod.yml` (new — PostgreSQL ile)

#### Dependencies

- T101

#### Success Criteria

- [ ] `docker compose up` ile çalışıyor
- [ ] Volume'lar persistent
- [ ] PostgreSQL sidecar opsiyonel
- [ ] Environment variable'lar ile config

---

### T103: Cross-Platform Binary Build

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64) için cross-compilation. GitHub Actions release workflow.

#### Files to Touch

- `Makefile` (update — cross-compile targets)
- `.github/workflows/release.yml` (update)
- `scripts/build-all.sh` (new)

#### Dependencies

- T002, T004

#### Success Criteria

- [ ] 6 platform için binary build ediliyor
- [ ] Binary'ler tag push'da otomatik release ediliyor
- [ ] Checksum dosyaları üretiliyor

---

### T104: Systemd Service Kurulumu

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

`kantar install-service` komutu ile systemd unit file oluşturma, enable ve start etme.

#### Technical Details

```ini
[Unit]
Description=Kantar Package Registry
After=network.target

[Service]
Type=simple
User=kantar
ExecStart=/usr/local/bin/kantar serve
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

#### Files to Touch

- `cmd/kantar/service.go` (new)
- `internal/service/systemd.go` (new)

#### Dependencies

- T092 (system komutları)

#### Success Criteria

- [ ] Unit file oluşturuluyor
- [ ] Service enable ve start ediliyor
- [ ] `kantar` user otomatik oluşturuluyor
- [ ] Log'lar journald'ye gidiyor

---

### T105: Helm Chart

**Status:** NOT_STARTED
**Priority:** P3
**Estimated Effort:** 2 days

#### Description

Kubernetes deploy için Helm chart. Deployment, Service, Ingress, PVC, ConfigMap, Secret.

#### Files to Touch

- `deploy/helm/kantar/Chart.yaml` (new)
- `deploy/helm/kantar/values.yaml` (new)
- `deploy/helm/kantar/templates/deployment.yaml` (new)
- `deploy/helm/kantar/templates/service.yaml` (new)
- `deploy/helm/kantar/templates/ingress.yaml` (new)
- `deploy/helm/kantar/templates/configmap.yaml` (new)
- `deploy/helm/kantar/templates/pvc.yaml` (new)

#### Dependencies

- T101

#### Success Criteria

- [ ] `helm install` çalışıyor
- [ ] values.yaml ile konfigüre edilebilir
- [ ] Ingress yapılandırması çalışıyor
- [ ] PVC ile persistent storage

---

### T106: Backup/Restore Mekanizması

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1.5 days

#### Description

Tam sistem backup (DB + storage + config) ve restore. tar.gz formatında arşiv.

#### Files to Touch

- `internal/backup/backup.go` (new)
- `internal/backup/restore.go` (new)
- `internal/backup/backup_test.go` (new)

#### Dependencies

- T010 (DB), T027 (storage)

#### Success Criteria

- [ ] Backup tar.gz üretiyor (DB + dosyalar + config)
- [ ] Restore backup'tan tam sistemi geri yüklüyor
- [ ] Backup sırasında veri tutarlılığı (DB snapshot)
- [ ] İlerleme raporlama

## Risk Assessment

- **Düşük Risk:** Standart deployment pattern'ları
- **Dikkat:** Cross-compilation'da CGo bağımlılıkları (SQLite) sorun çıkarabilir
- **Çözüm:** modernc.org/sqlite (pure Go SQLite) kullanılabilir veya CGo cross-compile toolchain
