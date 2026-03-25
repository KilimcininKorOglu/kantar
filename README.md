# Kantar

Unified local package registry platform for enterprise environments.

Kantar proxies, mirrors, approves, and serves packages from multiple ecosystems behind a corporate firewall. It replaces heavyweight solutions like JFrog Artifactory and Sonatype Nexus with a single lightweight Go binary.

*Tartilmis, olculmus, onaylanmis.* (Weighed, measured, approved.)

## Features

- **8 Package Ecosystems** -- Docker, npm, PyPI, Go Modules, Cargo, Maven, NuGet, Helm
- **Allowlist / Mirror Modes** -- explicit approval (default) or pull-through with blocklist
- **Policy Engine** -- declarative rules for license, vulnerability severity, package age, size, and naming
- **RBAC** -- 5 roles: Super Admin, Registry Admin, Publisher, Consumer, Viewer
- **Audit Trail** -- hash-chain tamper-evident logging
- **Web Dashboard** -- embedded React SPA with user management, package approval, audit viewer
- **CLI Tool** -- `kantarctl` for scripting and automation
- **Single Binary** -- Go binary with embedded web UI, no separate frontend deployment

## Quick Start

### Docker Compose (Recommended)

```bash
git clone https://github.com/KilimcininKorOglu/kantar.git
cd kantar
docker compose up --build -d
```

On first run, Kantar creates a default admin user and prints the password to stdout:

```bash
docker compose logs kantar | grep password
```

Open http://localhost:8080 and sign in with `admin` and the printed password.

### From Source

Prerequisites: Go 1.26+, Node.js 22+, PostgreSQL

```bash
# Build web UI
cd web && npm install && npm run build && cd ..

# Build binaries
make build-all

# Initialize config
./bin/kantar init

# Edit kantar.toml to configure PostgreSQL connection, then:
./bin/kantar serve
```

## Architecture

```
                     +------------------+
                     |   Web Browser    |
                     +--------+---------+
                              |
                     +--------v---------+
                     |   chi Router     |
                     |   (port 8080)    |
                     +--------+---------+
                              |
          +-------------------+-------------------+
          |                   |                   |
  +-------v-------+  +-------v-------+  +-------v-------+
  | /api/v1/*     |  | /{ecosystem}/ |  | /* (SPA)      |
  | Management    |  | Plugin Routes |  | Embedded      |
  | REST API      |  | (npm, docker, |  | React UI      |
  |               |  |  pypi, etc.)  |  |               |
  +-------+-------+  +-------+-------+  +---------------+
          |                   |
  +-------v-------+  +-------v-------+
  | Auth / RBAC   |  | RegistryPlugin|
  | JWT + bcrypt  |  | Interface     |
  +-------+-------+  +-------+-------+
          |                   |
  +-------v-------------------v-------+
  |        Core Engine                |
  |  Package Manager | Policy Engine  |
  |  Audit Logger    | Cache Layer    |
  +-------+---------------------------+
          |
  +-------v-------+  +---------------+
  | PostgreSQL    |  | Filesystem    |
  | (database)    |  | (storage)     |
  +---------------+  +---------------+
```

### Plugin System

Each ecosystem is a compile-time Go plugin implementing the `RegistryPlugin` interface. Plugins serve native protocol endpoints (Docker Registry API v2, npm registry API, PyPI Simple API, etc.) under `/{ecosystem}/` routes.

### Operation Modes

| Mode      | Behavior                                                      |
|-----------|---------------------------------------------------------------|
| Allowlist | Only explicitly approved packages can be pulled (default)     |
| Mirror    | All packages flow through; blocklist for exclusions           |

## Configuration

Kantar uses TOML configuration with environment variable interpolation:

```toml
[server]
host = "0.0.0.0"
port = 8080

[database]
type = "postgres"

[database.postgres]
host = "localhost"
port = 5432
name = "kantar"
user = "kantar"
password = "${KANTAR_DB_PASSWORD}"

[registry.npm]
mode = "allowlist"
upstream = "https://registry.npmjs.org"
enabled = true

[registry.docker]
mode = "allowlist"
upstream = "https://registry-1.docker.io"
enabled = true
```

## API

### Management API (`/api/v1`)

| Method | Endpoint                             | Auth         | Description              |
|--------|--------------------------------------|--------------|--------------------------|
| POST   | `/auth/login`                        | Public       | Get JWT token            |
| POST   | `/auth/register`                     | Public       | Create user              |
| GET    | `/system/status`                     | Any role     | Runtime info             |
| GET    | `/users`                             | Super Admin  | List users               |
| PUT    | `/users/{id}`                        | Super Admin  | Update user              |
| DELETE | `/users/{id}`                        | Super Admin  | Delete user              |
| GET    | `/packages?registry=npm&status=pending` | Consumer+ | List packages         |
| POST   | `/packages/{id}/approve`             | Reg. Admin+  | Approve package          |
| POST   | `/packages/{id}/block`               | Reg. Admin+  | Block package            |
| GET    | `/audit`                             | Reg. Admin+  | Audit log entries        |
| GET    | `/audit/verify`                      | Reg. Admin+  | Verify hash chain        |

### Native Protocol Endpoints

| Ecosystem  | Prefix     | Protocol                       |
|------------|------------|--------------------------------|
| Docker     | `/docker/` | Docker Registry API v2         |
| npm        | `/npm/`    | npm Registry API               |
| PyPI       | `/pypi/`   | PEP 503 Simple API             |
| Go Modules | `/gomod/`  | Go Module Proxy Protocol       |
| Cargo      | `/cargo/`  | Sparse Registry (RFC 2789)     |
| Maven      | `/maven/`  | Maven Repository Layout        |
| NuGet      | `/nuget/`  | NuGet V3 API                   |
| Helm       | `/helm/`   | Helm Chart Repository          |

## Development

```bash
make build-all              # Build server + CLI
make test                   # Run tests with race detector
make lint                   # golangci-lint
make fmt                    # Format code
sqlc generate               # Regenerate SQL query code
cd web && npm run dev       # Frontend dev server with HMR
```

### Project Structure

```
cmd/kantar/          Server binary (serve, init, version)
cmd/kantarctl/       CLI tool
internal/server/     HTTP server, middleware, API handlers
internal/auth/       JWT, bcrypt, RBAC, API tokens
internal/database/   PostgreSQL, migrations, sqlc queries
internal/storage/    Filesystem with atomic writes
internal/cache/      In-memory LRU cache
internal/manager/    Package lifecycle management
internal/audit/      Hash-chain audit logging
internal/policy/     Policy engine (license, size, age, version)
internal/plugin/     Plugin registry and route mounting
internal/plugins/    8 ecosystem plugin implementations
internal/config/     TOML config loader with env interpolation
pkg/registry/        Public RegistryPlugin interface
web/                 React 19 + Vite 6 + Tailwind 4 SPA
migrations/          Embedded PostgreSQL schema
deploy/helm/         Kubernetes Helm chart
```

## Deployment

### Docker Compose

```bash
docker compose up --build -d
```

Uses PostgreSQL with bind-mount volumes at `./docker-data/`. Config file `kantar.toml` is mounted as `/etc/kantar/kantar.toml`.

### Kubernetes

```bash
helm install kantar deploy/helm/kantar/ \
  --set database.postgres.host=postgres.svc \
  --set database.postgres.password=secret
```

### Binary

Download from [Releases](https://github.com/KilimcininKorOglu/kantar/releases). Available for Linux, macOS, and Windows (amd64/arm64).

```bash
kantar init              # Generate default config
kantar serve --config kantar.toml
```

## License

This project is proprietary software.
