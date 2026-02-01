# Kantar - Unified Local Package Registry Platform

## Product Requirements Document (PRD)

**Version:** 1.0
**Date:** 2026-03-31
**Status:** Draft

---

## 1. Executive Summary

Kantar, kurumsal ortamlar icin gelistirilmis, tek bir platform uzerinden Docker, npm, PyPI, Go Modules ve Cargo (Rust) paket ekosistemlerini yoneten bir unified local package registry sistemidir. Sistem yoneticileri guvenli buldugu paketleri versiyonlari ile local registery'ye ceker, kurumdaki tum gelistiriciler yalnizca bu onaylanmis paketleri kullanir. Kurum ici private paketler de ayni platform uzerinden yayinlanabilir.

Kantar, JFrog Artifactory ve Sonatype Nexus gibi agirsiklet enterprise cozumlerine hafif, kolay kurulan, plugin-tabanli bir alternatif sunar. Go ile gelistirilir, tek binary olarak dagilir ve minimum bagimlilikla calisir.

---

## 2. Problem Statement

### Mevcut Sorunlar

- **Guvenlik riski:** Gelistiricilerin dogrudan public registry'lerden paket cekmesi supply-chain attack'lara acik kapidir (bkz: event-stream, ua-parser-js, colors.js incidents)
- **Tutarsizlik:** Farkli gelistiricilerin farkli versiyonlari kullanmasi reproducibility sorunlarina yol acar
- **Ag bagimliligi:** Her build'de public registry'lere erisim gerekir, ag kesintilerinde is durur
- **Denetim eksikligi:** Hangi paketin kim tarafindan ne zaman cekildiginin takibi yapilmaz
- **Dagnik cozumler:** Her ekosistem icin ayri registry (Verdaccio, Harbor, devpi...) kurmak operasyonel yuk olusturur
- **Maliyet:** Artifactory/Nexus Pro lisanslari kucuk-orta olcekli kurumlar icin pahalidir

### Kantar'in Cozumu

Tek binary, tek UI, tek API — tum paket ekosistemlerini unified bir katman altinda yonetir. Admin onay mekanizmasi ile yalnizca guvenilir paketler kurum agina alinir.

---

## 3. Target Users

| Rol | Ihtiyac |
|-----|---------|
| **Sistem Yoneticisi / DevOps** | Paket onaylama, versiyon kontrolu, guvenlik taramasi, kullanici yonetimi |
| **Gelistirici** | Hizli paket indirme, private paket yayinlama, mevcut toolchain uyumu |
| **Guvenlik Ekibi** | Audit log, vulnerability scanning entegrasyonu, policy enforcement |
| **Takim Lideri** | Takim bazli erisim kontrolu, paket kullanim raporu |

---

## 4. Core Architecture

### 4.1 High-Level Architecture

```
+------------------------------------------------------------------+
|                        KANTAR PLATFORM                           |
|                                                                  |
|  +-------------------+    +------------------+                   |
|  |    Web UI (SPA)    |    |   REST API       |                   |
|  |  Admin Dashboard   |    |   /api/v1/*      |                   |
|  +--------+----------+    +--------+---------+                   |
|           |                        |                             |
|  +--------+------------------------+---------+                   |
|  |              Core Engine                   |                  |
|  |                                            |                  |
|  |  +----------+  +----------+  +----------+  |                  |
|  |  | Auth &   |  | Package  |  | Policy   |  |                  |
|  |  | RBAC     |  | Manager  |  | Engine   |  |                  |
|  |  +----------+  +----------+  +----------+  |                  |
|  |                                            |                  |
|  |  +----------+  +----------+  +----------+  |                  |
|  |  | Storage  |  | Cache    |  | Audit    |  |                  |
|  |  | Layer    |  | Layer    |  | Logger   |  |                  |
|  |  +----------+  +----------+  +----------+  |                  |
|  +--------------------------------------------+                  |
|                                                                  |
|  +--------------------------------------------+                  |
|  |           Plugin Registry                   |                 |
|  |                                             |                 |
|  |  +--------+ +--------+ +--------+          |                 |
|  |  | Docker | | npm    | | PyPI   |          |                 |
|  |  | Plugin | | Plugin | | Plugin |          |                 |
|  |  +--------+ +--------+ +--------+          |                 |
|  |  +--------+ +--------+ +--------+          |                 |
|  |  | Go Mod | | Cargo  | | Custom |          |                 |
|  |  | Plugin | | Plugin | | Plugin |          |                 |
|  |  +--------+ +--------+ +--------+          |                 |
|  +--------------------------------------------+                  |
|                                                                  |
|  +--------------------------------------------+                  |
|  |           Storage Backend                   |                 |
|  |  Local FS | S3/MinIO | SQLite/PostgreSQL   |                 |
|  +--------------------------------------------+                  |
+------------------------------------------------------------------+
         |              |              |
   +-----+----+  +-----+----+  +------+-----+
   | Upstream  |  | Upstream |  | Upstream   |
   | Docker    |  | npmjs   |  | pypi.org   |
   | Hub       |  | .com    |  |            |
   +-----------+  +---------+  +------------+
```

### 4.2 Plugin Architecture

Kantar'in kalbi plugin sistemidir. Her paket ekosistemi bir plugin olarak implemente edilir. Bu sayede:

- Core engine ekosistemden bagimsizdir
- Yeni ekosistemler plugin yazarak eklenir (Maven, NuGet, Helm, etc.)
- Community plugin destegi saglanir
- Her plugin bagimsiz olarak guncellenebilir

```
Plugin Interface (Go):

type RegistryPlugin interface {
    // Metadata
    Name() string
    Version() string
    EcosystemType() EcosystemType  // Docker, NPM, PyPI, GoMod, Cargo

    // Upstream Operations
    Search(ctx context.Context, query string) ([]PackageMeta, error)
    FetchPackage(ctx context.Context, name, version string) (*Package, error)
    FetchMetadata(ctx context.Context, name string) (*PackageMeta, error)

    // Local Registry Operations
    ServePackage(w http.ResponseWriter, r *http.Request)  // Native protocol handler
    PublishPackage(ctx context.Context, pkg *Package) error
    DeletePackage(ctx context.Context, name, version string) error

    // Validation
    ValidatePackage(ctx context.Context, pkg *Package) (*ValidationResult, error)

    // Configuration
    Configure(config map[string]interface{}) error
    DefaultConfig() map[string]interface{}

    // Protocol-specific routes
    Routes() []Route
}
```

### 4.3 Storage Layout

```
/var/lib/kantar/
  +-- data/
  |   +-- docker/
  |   |   +-- registry/          # OCI layout
  |   |   +-- blobs/
  |   +-- npm/
  |   |   +-- packages/
  |   |   +-- tarballs/
  |   +-- pypi/
  |   |   +-- packages/
  |   |   +-- wheels/
  |   +-- gomod/
  |   |   +-- modules/
  |   |   +-- cache/
  |   +-- cargo/
  |       +-- crates/
  |       +-- index/
  +-- db/
  |   +-- kantar.db              # SQLite (default)
  +-- config/
  |   +-- kantar.toml            # Main config
  |   +-- policies/              # Policy files
  +-- plugins/
  |   +-- docker.so              # Plugin binaries (veya built-in)
  |   +-- npm.so
  +-- logs/
  |   +-- audit.log
  |   +-- access.log
  +-- cache/
      +-- upstream/              # Upstream response cache
```

---

## 5. Feature Specifications

### 5.1 Package Lifecycle Management

#### 5.1.1 Upstream Mirroring (Pull-Through Cache)

Admin'in onayladigi paketler upstream registry'den cekilir ve lokal'de depolanir.

**Iki mod desteklenir:**

1. **Allowlist Mode (Varsayilan):** Yalnizca acikca onaylanmis paketler cekilebilir
2. **Mirror Mode:** Tum paketler cekilebilir, blocklist ile engelleme yapilir

```
# kantar.toml ornek yapilandirma

[registry.npm]
mode = "allowlist"             # allowlist | mirror
upstream = "https://registry.npmjs.org"
auto_sync = true               # Yeni versiyonlar otomatik cekilsin mi?
auto_sync_interval = "6h"      # Otomatik senkron araligi
max_versions = 0               # 0 = hepsi, N = son N versiyon

[registry.npm.allowlist]
packages = [
    { name = "express", versions = ["4.*", "5.*"] },
    { name = "lodash", versions = ["*"] },
    { name = "react", versions = ["18.*", "19.*"] },
    { name = "@types/*", versions = ["*"] },      # Glob destegi
]

[registry.docker]
mode = "allowlist"
upstream = "https://registry-1.docker.io"

[registry.docker.allowlist]
images = [
    { name = "library/node", tags = ["20-alpine", "22-alpine"] },
    { name = "library/postgres", tags = ["16.*", "17.*"] },
    { name = "mycompany/*", tags = ["*"] },        # Org-level glob
]
```

#### 5.1.2 Private Package Publishing

Kurum ici paketler standart toolchain ile publish edilir:

```bash
# npm
npm publish --registry http://kantar.local:8080/npm/

# pip
twine upload --repository-url http://kantar.local:8080/pypi/ dist/*

# docker
docker push kantar.local:8080/internal/my-service:1.0

# go
GOPROXY=http://kantar.local:8080/go go publish  # go mod ile

# cargo
cargo publish --registry kantar --token $KANTAR_TOKEN
```

#### 5.1.3 Package Approval Workflow

```
Developer/Admin           Kantar                Upstream
     |                      |                      |
     |-- Request Package -->|                      |
     |                      |-- Fetch Metadata --->|
     |                      |<-- Package Info -----|
     |                      |                      |
     |                      |-- Policy Check       |
     |                      |-- Vuln Scan (opt.)   |
     |                      |-- License Check      |
     |                      |                      |
     |<-- Approval Needed --|  (if manual approval required)
     |                      |
     |-- Approve ---------->|
     |                      |-- Download Package ->|
     |                      |<-- Package Data -----|
     |                      |-- Store Locally      |
     |<-- Available --------|
```

### 5.2 Access Control & Authentication

#### 5.2.1 Authentication Methods

- **Local accounts:** Built-in kullanici yonetimi (varsayilan)
- **LDAP/Active Directory:** Kurumsal dizin entegrasyonu
- **OIDC/OAuth2:** SSO desteği (Keycloak, Azure AD, Okta, etc.)
- **API Token:** CI/CD ve otomasyon icin token-based auth

#### 5.2.2 RBAC Model

```
Roller:
  +-- Super Admin
  |     Tum sistem ayarlari, kullanici yonetimi, plugin yonetimi
  |
  +-- Registry Admin
  |     Belirli registry'ler icin paket onaylama, policy yonetimi
  |
  +-- Publisher
  |     Private paket yayinlama (atandigi namespace'lerde)
  |
  +-- Consumer (Varsayilan)
  |     Onaylanmis paketleri indirme
  |
  +-- Viewer
        Salt okunur erisim, paket metadata goruntuleme

Namespace-based Scope:
  - Admin, rolleri namespace (org/team) bazinda atayabilir
  - Ornek: "Ali, @frontend-team/* npm paketleri icin Publisher"
```

### 5.3 Policy Engine

Kurumsal kurallar deklaratif olarak tanimlanir:

```toml
# policies/security.toml

[policy.license]
allowed = ["MIT", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "ISC"]
blocked = ["GPL-3.0", "AGPL-3.0"]  # Ticari yazilim icin
action = "block"                    # block | warn | log

[policy.vulnerability]
block_severity = ["critical", "high"]
warn_severity = ["medium"]
allow_severity = ["low"]
scanner = "grype"                   # grype | trivy | none
auto_scan = true
scan_on_sync = true

[policy.age]
min_package_age = "7d"              # Typosquatting koruması - yeni paketleri beklet
min_maintainers = 2                 # Tek kisilik paketleri engelle (opsiyonel)

[policy.size]
max_package_size = "500MB"          # Docker image'lar icin daha yuksek
max_layer_count = 20                # Docker specific

[policy.naming]
blocked_scopes = ["@evil-corp"]     # npm scope block
blocked_prefixes = ["__test"]

[policy.version]
pin_strategy = "minor"              # major | minor | patch | exact
allow_prerelease = false
allow_deprecated = false
```

### 5.4 Web UI

#### 5.4.1 Admin Dashboard

```
+--------------------------------------------------------------+
| KANTAR                              admin@company  [Logout]  |
+------+-------------------------------------------------------+
|      |                                                       |
| NAV  |  Dashboard                                            |
|      |  +---------------------------------------------------+|
| [ ] Overview   |  Packages: 1,247  |  Storage: 48.2 GB     ||
| [ ] Packages   |  Pending:  12     |  Users: 89            ||
| [ ] Registries |  Downloads: 4.2K  |  Uptime: 99.97%       ||
| [ ] Users      |  (today)          |                        ||
| [ ] Policies   +---------------------------------------------------+|
| [ ] Audit Log  |                                            ||
| [ ] Settings   |  Registry Health                           ||
|      |  +----------+----------+----------+----------+------+||
|      |  | Docker   | npm      | PyPI     | Go Mod   |Cargo |||
|      |  | OK  324  | OK  512  | OK  198  | OK  145  |OK 68 |||
|      |  | 12.1 GB  | 8.4 GB   | 6.2 GB   | 2.8 GB   |1.1GB|||
|      |  +----------+----------+----------+----------+------+||
|      |                                                       ||
|      |  Pending Approvals                                    ||
|      |  +---------------------------------------------------+||
|      |  | Package         | Ver   | Registry | Requested By |||
|      |  |-----------------|-------|----------|------------- |||
|      |  | axios           | 1.7.2 | npm      | dev-team     |||
|      |  | fastapi         | 0.115 | pypi     | ml-team      |||
|      |  | redis:7-alpine  | 7.4   | docker   | ops-team     |||
|      |  +---------------------------------------------------+||
|      |                                                       ||
|      |  Recent Activity                                      ||
|      |  +---------------------------------------------------+||
|      |  | 14:22 ali.yilmaz downloaded express@4.18.2 (npm)  |||
|      |  | 14:18 ci-runner pulled node:20-alpine (docker)    |||
|      |  | 14:15 admin approved lodash@4.17.21 (npm)         |||
|      |  | 14:10 zeynep.k published @internal/auth@2.1 (npm) |||
|      |  +---------------------------------------------------+||
+------+-------------------------------------------------------+
```

#### 5.4.2 Package Detail View

```
+--------------------------------------------------------------+
| KANTAR  > npm > express                                      |
+------+-------------------------------------------------------+
|      |                                                       |
| NAV  |  express                           [Approved] [Sync] |
|      |  Fast, unopinionated web framework for Node.js        |
|      |                                                       |
|      |  +-- Versions ----+-- Info ----------+-- Stats ------+|
|      |  | v4.21.2  [cur] | License: MIT     | Downloads:    ||
|      |  | v4.21.1        | Size: 214 KB     | Today: 47     ||
|      |  | v4.21.0        | Deps: 31         | Week: 312     ||
|      |  | v4.20.0        | Vulns: 0         | Total: 8,941  ||
|      |  | v5.0.1  [beta] | Synced: 2h ago   |               ||
|      |  +----------------+------------------+---------------+|
|      |                                                       |
|      |  Dependencies (showing direct, 31 total)              |
|      |  +---------------------------------------------------+|
|      |  | accepts@1.3.8 [ok] | body-parser@1.20.3 [ok] |   ||
|      |  | content-disp.@0.5  | cookie@0.7.1 [ok]       |   ||
|      |  | ...                                               ||
|      |  +---------------------------------------------------+|
|      |                                                       |
|      |  Vulnerability Report                                 |
|      |  +---------------------------------------------------+|
|      |  | No known vulnerabilities                    [Scan] ||
|      |  | Last scanned: 2026-03-31 12:00 UTC                ||
|      |  +---------------------------------------------------+|
+------+-------------------------------------------------------+
```

### 5.5 CLI Tool (kantarctl)

```bash
# Registry yonetimi
kantarctl registry list
kantarctl registry add npm --upstream https://registry.npmjs.org
kantarctl registry sync npm                    # Tum allowlist'i senkronla
kantarctl registry sync npm --package express  # Tek paket senkronla

# Paket yonetimi
kantarctl package search express --registry npm
kantarctl package approve express@4.21.2 --registry npm
kantarctl package approve "express@4.*" --registry npm  # Glob
kantarctl package block malicious-pkg --registry npm --reason "supply-chain risk"
kantarctl package info express --registry npm
kantarctl package deps express@4.21.2 --registry npm --tree  # Dependency tree

# Bulk islemler
kantarctl package import --file approved-packages.toml  # Toplu onaylama
kantarctl package export --registry npm --format toml   # Mevcut listeyi export

# Policy yonetimi
kantarctl policy validate                      # Policy dosyalarini dogrula
kantarctl policy test express@4.21.2 --registry npm  # Paketi policy'ye karsi test et

# Kullanici yonetimi
kantarctl user list
kantarctl user create --username ali --role consumer
kantarctl user token create --username ci-runner --expires 90d

# Sistem
kantarctl status                               # Tum registry'lerin durumu
kantarctl gc                                   # Garbage collection (eski versiyonlar)
kantarctl backup --output /backup/kantar.tar
kantarctl restore --input /backup/kantar.tar
```

### 5.6 Client Configuration

Gelistiricilerin mevcut toolchain'lerini Kantar'a yonlendirmesi:

```bash
# --- npm ---
npm config set registry http://kantar.local:8080/npm/
# veya .npmrc
registry=http://kantar.local:8080/npm/
//kantar.local:8080/npm/:_authToken=${KANTAR_TOKEN}

# --- pip ---
pip install --index-url http://kantar.local:8080/pypi/simple/ flask
# veya pip.conf / pip.ini
[global]
index-url = http://kantar.local:8080/pypi/simple/
trusted-host = kantar.local

# --- Docker ---
# /etc/docker/daemon.json
{
    "registry-mirrors": ["http://kantar.local:8080"],
    "insecure-registries": ["kantar.local:8080"]
}

# --- Go ---
export GOPROXY=http://kantar.local:8080/go,direct
export GONOSUMDB=company.internal/*
export GONOSUMCHECK=company.internal/*

# --- Cargo ---
# ~/.cargo/config.toml
[registries.kantar]
index = "sparse+http://kantar.local:8080/cargo/"
token = "Bearer ${KANTAR_TOKEN}"

[source.crates-io]
replace-with = "kantar"

[source.kantar]
registry = "sparse+http://kantar.local:8080/cargo/"
```

### 5.7 Audit & Logging

Tum islemler audit log'a kaydedilir:

```json
{
    "timestamp": "2026-03-31T14:22:33Z",
    "event": "package.download",
    "actor": {
        "username": "ali.yilmaz",
        "ip": "10.0.1.45",
        "user_agent": "npm/10.8.0"
    },
    "resource": {
        "registry": "npm",
        "package": "express",
        "version": "4.21.2"
    },
    "result": "success",
    "metadata": {
        "response_time_ms": 12,
        "cache_hit": true,
        "bytes_transferred": 219648
    }
}
```

**Desteklenen event turleri:**
- `package.download` / `package.upload` / `package.delete`
- `package.approve` / `package.block`
- `policy.violation` / `policy.update`
- `user.login` / `user.create` / `user.token.create`
- `registry.sync` / `registry.config.update`
- `system.gc` / `system.backup`

### 5.8 High Availability & Replication (v2 - Plugin)

Ilk surumde single-instance, ilerleyen surumlerde:

- **Active-Passive:** PostgreSQL replication + shared storage (S3/MinIO)
- **Multi-node:** Consistent hashing ile paket dagitimi
- **CDN Mode:** Edge node'lar ile cografi yakinlik

---

## 6. API Design

### 6.1 Management API (REST)

```
Base: /api/v1

# Registries
GET     /registries                          # Tum registry'leri listele
GET     /registries/{type}                   # Registry detayi (npm, docker, etc.)
PUT     /registries/{type}/config            # Registry yapilandirmasi

# Packages
GET     /registries/{type}/packages          # Paketleri listele (?search=&status=)
GET     /registries/{type}/packages/{name}   # Paket detayi
POST    /registries/{type}/packages/{name}/approve    # Onayla
POST    /registries/{type}/packages/{name}/block      # Engelle
DELETE  /registries/{type}/packages/{name}/{version}  # Versiyonu sil

# Sync
POST    /registries/{type}/sync              # Tam senkronizasyon
POST    /registries/{type}/sync/{name}       # Tek paket senkronla

# Users
GET     /users
POST    /users
PUT     /users/{id}
DELETE  /users/{id}
POST    /users/{id}/tokens                   # API token olustur

# Policies
GET     /policies
PUT     /policies/{name}
POST    /policies/validate                   # Policy test

# Audit
GET     /audit                               # ?actor=&event=&from=&to=
GET     /audit/export                        # CSV/JSON export

# System
GET     /system/status                       # Health check
POST    /system/gc                           # Garbage collection
POST    /system/backup
```

### 6.2 Native Protocol Endpoints

Her plugin kendi native protokolunu serve eder:

```
# Docker Registry API v2
/v2/                                         # API version check
/v2/{name}/manifests/{reference}             # Image manifest
/v2/{name}/blobs/{digest}                    # Layer blob
/v2/{name}/blobs/uploads/                    # Layer upload

# npm Registry API
/npm/{package}                               # Package metadata
/npm/{package}/-/{tarball}                   # Package tarball
/npm/-/user/org.couchdb.user:{user}          # Auth

# PyPI Simple API (PEP 503)
/pypi/simple/                                # Package index
/pypi/simple/{package}/                      # Package versions
/pypi/packages/{path}                        # Package file

# Go Module Proxy (GOPROXY protocol)
/go/{module}/@v/list                         # Version list
/go/{module}/@v/{version}.info               # Version info
/go/{module}/@v/{version}.mod                # go.mod file
/go/{module}/@v/{version}.zip                # Module zip

# Cargo Sparse Registry (RFC 2789)
/cargo/config.json                           # Registry config
/cargo/{prefix}/{crate}                      # Crate metadata
/cargo/api/v1/crates                         # API endpoint
/cargo/api/v1/crates/new                     # Publish
```

---

## 7. Configuration

### 7.1 Main Configuration File

```toml
# /etc/kantar/kantar.toml

[server]
host = "0.0.0.0"
port = 8080
tls_cert = ""                      # Bos ise HTTP, dolu ise HTTPS
tls_key = ""
base_url = "https://kantar.company.internal"
workers = 0                        # 0 = CPU sayisi kadar

[storage]
type = "filesystem"                # filesystem | s3
path = "/var/lib/kantar/data"

[storage.s3]                       # type = "s3" ise
endpoint = "https://minio.local:9000"
bucket = "kantar"
access_key = "${KANTAR_S3_ACCESS_KEY}"
secret_key = "${KANTAR_S3_SECRET_KEY}"
region = "us-east-1"

[database]
type = "sqlite"                    # sqlite | postgres
path = "/var/lib/kantar/db/kantar.db"   # sqlite icin

[database.postgres]                # type = "postgres" ise
host = "localhost"
port = 5432
name = "kantar"
user = "kantar"
password = "${KANTAR_DB_PASSWORD}"
ssl_mode = "require"

[auth]
type = "local"                     # local | ldap | oidc
session_ttl = "24h"
token_ttl = "90d"

[auth.ldap]                        # type = "ldap" ise
url = "ldap://ldap.company.internal:389"
base_dn = "dc=company,dc=internal"
bind_dn = "cn=kantar,dc=company,dc=internal"
bind_password = "${KANTAR_LDAP_PASSWORD}"
user_filter = "(uid={username})"
group_filter = "(member={dn})"

[auth.oidc]                        # type = "oidc" ise
issuer = "https://keycloak.company.internal/realms/company"
client_id = "kantar"
client_secret = "${KANTAR_OIDC_SECRET}"
scopes = ["openid", "profile", "email"]

[cache]
enabled = true
type = "memory"                    # memory | redis
max_size = "2GB"                   # memory cache icin
ttl = "1h"                         # Upstream metadata cache TTL

[cache.redis]                      # type = "redis" ise
addr = "localhost:6379"
password = ""
db = 0

[logging]
level = "info"                     # debug | info | warn | error
format = "json"                    # json | text
audit_enabled = true
audit_path = "/var/lib/kantar/logs/audit.log"
audit_retention = "365d"

[notifications]
enabled = false

[notifications.webhook]
url = "https://hooks.slack.com/services/..."
events = ["package.approve", "policy.violation", "system.error"]

[notifications.email]
smtp_host = "smtp.company.internal"
smtp_port = 587
from = "kantar@company.internal"
to = ["devops@company.internal"]
events = ["policy.violation", "system.error"]
```

---

## 8. Deployment

### 8.1 Single Binary

```bash
# Download
curl -fsSL https://github.com/user/kantar/releases/latest/download/kantar-linux-amd64 \
    -o /usr/local/bin/kantar
chmod +x /usr/local/bin/kantar

# Initialize
kantar init                        # Default config olusturur
kantar serve                       # Servisi baslat

# Systemd service
kantar install-service             # systemd unit file olusturur ve enable eder
```

### 8.2 Docker Compose

```yaml
# docker-compose.yml
version: "3.8"

services:
    kantar:
        image: ghcr.io/user/kantar:latest
        ports:
            - "8080:8080"
        volumes:
            - kantar-data:/var/lib/kantar
            - ./kantar.toml:/etc/kantar/kantar.toml:ro
            - ./policies:/etc/kantar/policies:ro
        environment:
            - KANTAR_DB_PASSWORD=${DB_PASSWORD}
        restart: unless-stopped

    # Opsiyonel: PostgreSQL
    postgres:
        image: postgres:17-alpine
        volumes:
            - pg-data:/var/lib/postgresql/data
        environment:
            - POSTGRES_DB=kantar
            - POSTGRES_USER=kantar
            - POSTGRES_PASSWORD=${DB_PASSWORD}
        restart: unless-stopped

volumes:
    kantar-data:
    pg-data:
```

### 8.3 Kubernetes (Helm)

```bash
helm repo add kantar https://charts.kantar.dev
helm install kantar kantar/kantar \
    --set persistence.size=100Gi \
    --set ingress.enabled=true \
    --set ingress.host=kantar.company.internal
```

---

## 9. Tech Stack

| Katman | Teknoloji |
|--------|-----------|
| **Dil** | Go 1.23+ |
| **HTTP Router** | net/http (stdlib) + chi router |
| **Database** | SQLite (default) / PostgreSQL (production) |
| **ORM/Query** | sqlc (type-safe SQL) |
| **Cache** | In-memory (default) / Redis |
| **Storage** | Local filesystem / S3-compatible (MinIO) |
| **CLI** | cobra + lipgloss (CLI output styling) |
| **Web UI** | Embedded SPA (Go embed) — React + Vite |
| **Auth** | JWT tokens + bcrypt |
| **Plugin System** | Go plugin interface (compile-time) |
| **Container** | Docker multi-stage build |
| **CI/CD** | GitHub Actions |
| **Vulnerability Scanner** | Grype / Trivy (external, opsiyonel) |

---

## 10. Security Considerations

### 10.1 Supply Chain Security

- **Checksum verification:** Upstream'den indirilen her paket checksum ile dogrulanir
- **Signature verification:** npm (npmjs signatures), Docker (Notary/cosign), Go (sumdb) destegi
- **Dependency resolution:** Bagimliliklari recursive olarak kontrol, onaylanmamis bagimliligi olan paket uyarisi
- **Typosquatting detection:** Mevcut populer paketlere benzeyen isimler icin uyari (Levenshtein distance)

### 10.2 Network Security

- TLS termination (built-in veya reverse proxy arkasinda)
- Rate limiting (per-user, per-IP)
- IP allowlist destegi
- CORS yapilandirmasi

### 10.3 Data Security

- Sifrelenmis storage destegi (at-rest encryption)
- Token'lar bcrypt ile hash'lenir
- Audit log tamper detection (hash chain)
- Secrets environment variable ile inject edilir (config dosyasinda plaintext yok)

---

## 11. Versioning & Roadmap

### v1.0 — Foundation

- [x] Core engine + plugin architecture
- [x] npm, PyPI, Docker, Go Modules, Cargo plugin'leri
- [x] Maven/Gradle plugin
- [x] NuGet plugin
- [x] Helm chart registry plugin
- [x] OCI artifact destegi (generic OCI)
- [x] Allowlist/Mirror mode
- [x] Local auth + RBAC
- [x] Web UI (admin dashboard)
- [x] Dependency graph visualization (Web UI)
- [x] kantarctl CLI
- [x] SQLite + PostgreSQL storage
- [x] Redis cache backend
- [x] Audit logging
- [x] Signed audit log (blockchain-style hash chain)
- [x] Policy engine (license, vulnerability, age)
- [x] Bulk import/export
- [x] Client configuration dokumanlanmasi

### v1.1 — Enterprise

- [ ] LDAP / OIDC authentication
- [ ] S3/MinIO storage
- [ ] Webhook/email notifications
- [ ] Vulnerability scanner entegrasyonu (Grype/Trivy)

### v2.0 — Scale

- [ ] Multi-node replication
- [ ] Kubernetes operator
- [ ] SBOM (Software Bill of Materials) generation
- [ ] Plugin marketplace

---

## 12. Success Metrics

| Metrik | Hedef |
|--------|-------|
| Paket indirme suresi (cache hit) | < 50ms |
| Upstream senkronizasyon suresi | < 5s/paket |
| Web UI sayfa yuklenme | < 1s |
| Sistem memory kullanimi (1K paket) | < 256MB |
| Kurulum suresi (basic setup) | < 5 dakika |
| Plugin ekleme suresi (gelistirici) | < 1 gun |
| Zero-downtime upstream kesintilerde | %100 cache serve |

---

## 13. Non-Goals (v1)

- Paketlerin otomatik build edilmesi (CI/CD degil)
- Source code hosting (Git server degil)
- Cloud-hosted SaaS versiyonu
- GUI-based policy editor (TOML dosyasi yeterli)
- Multi-tenant (tek kurum icin tasarlanmis, v2'de degerlendirilecek)

---

## 14. Competitive Analysis

| Ozellik | Kantar | Artifactory | Nexus OSS | Verdaccio |
|---------|--------|-------------|-----------|-----------|
| Unified registries | Tumu | Tumu | Tumu | Yalnizca npm |
| Self-hosted | Evet | Evet | Evet | Evet |
| Tek binary | Evet | Hayir | Hayir | Hayir |
| Plugin sistemi | Evet | Evet | Limited | Evet |
| Kurulum kolayligi | < 5dk | Saatler | 30dk+ | < 5dk |
| License | OSS | Ticari | OSS/Pro | OSS |
| Allowlist mode | Native | Var | Var | Yok |
| Policy engine | Built-in | Xray (ayri) | IQ (ayri) | Yok |
| Memory footprint | ~256MB | ~4GB+ | ~2GB+ | ~128MB |
| Go native | Evet | Java | Java | Node.js |

---

*Kantar — Tartilmis, olculmus, onaylanmis.*
