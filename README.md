# Kantar

Unified local package registry platform for enterprise environments.

Kantar proxies, mirrors, approves, and serves packages from multiple ecosystems behind a corporate firewall. It replaces heavyweight solutions like JFrog Artifactory and Sonatype Nexus with a single lightweight Go binary.

*Trust nothing. Approve everything.*

## Features

- **8 Package Ecosystems** -- Docker, npm, PyPI, Go Modules, Cargo, Maven, NuGet, Helm
- **Allowlist / Mirror Modes** -- explicit approval (default) or pull-through with blocklist
- **Recursive Dependency Sync** -- approve a package and its entire dependency tree is auto-fetched
- **Policy Engine** -- declarative rules for license, vulnerability severity, package age, size, and naming
- **RBAC** -- 5 roles: Super Admin, Registry Admin, Publisher, Consumer, Viewer
- **Audit Trail** -- hash-chain tamper-evident logging
- **Web Dashboard** -- embedded React SPA with settings, registry, policy, and user management
- **Runtime Configuration** -- manage settings, registries, and policies via Web UI without restart
- **Multi-Language** -- English, Turkish, German; per-user language preference
- **Per-User Timezone** -- each user selects their timezone, all dates displayed accordingly
- **CLI Tool** -- `kantarctl` for scripting and automation
- **Single Binary** -- Go binary with embedded web UI, no separate frontend deployment

## Quick Start

### Docker Compose (Recommended)

```bash
git clone https://github.com/KilimcininKorOglu/kantar.git
cd kantar
make docker-up
```

On first run, Kantar creates a default admin user and prints the password to stdout:

```bash
make docker-logs | grep password
```

Open http://localhost:8080 and sign in with `admin` and the printed password.

### From Source

Prerequisites: Go 1.26+, Node.js 22+, PostgreSQL

```bash
# Build web UI + binaries
make web
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
  |  Audit Logger    | Sync Engine   |
  +-------+---------------------------+
          |
  +-------v-------+  +---------------+
  | PostgreSQL    |  | Filesystem    |
  | (database +   |  | (storage)     |
  |  settings)    |  |               |
  +---------------+  +---------------+
```

### Plugin System

Each ecosystem is a compile-time Go plugin implementing the `RegistryPlugin` interface. Plugins serve native protocol endpoints under `/{ecosystem}/` routes and implement `ResolveDependencies` for recursive sync.

### Recursive Dependency Sync

When a package is approved, Kantar automatically resolves and approves its entire dependency tree:

1. Admin approves `express` on npm
2. Sync engine fetches the packument from `registry.npmjs.org`
3. Semver resolver picks the best matching version for each dependency
4. BFS traversal processes all transitive dependencies (max depth 10)
5. Each dependency is auto-approved and recorded in the database

Supported ecosystems: npm, PyPI, Go Modules, Cargo, Maven, NuGet, Helm. Docker is excluded (no dependency concept).

### Operation Modes

| Mode      | Behavior                                                      |
|-----------|---------------------------------------------------------------|
| Allowlist | Only explicitly approved packages can be pulled (default)     |
| Mirror    | All packages flow through; blocklist for exclusions           |

## Configuration

Kantar uses a two-tier configuration model:

- **`kantar.toml`** -- bootstrap settings (server, storage, database, auth type)
- **Database** -- runtime settings managed via Web UI (logging, cache, registries, policies)

```toml
# kantar.toml -- Bootstrap Only
[server]
host = "0.0.0.0"
port = 8080

[storage]
type = "filesystem"
path = "/var/lib/kantar/data"

[database]
type = "postgres"

[database.postgres]
host = "localhost"
port = 5432
name = "kantar"
user = "kantar"
password = "${KANTAR_DB_PASSWORD}"
ssl_mode = "disable"

[auth]
type = "local"
```

Runtime settings (log level, cache TTL, session TTL, registry modes, policy rules) are seeded from defaults on first run and managed via the Settings, Registries, and Policies pages in the Web UI.

## API

### Management API (`/api/v1`)

| Method | Endpoint                                | Auth         | Description                |
|--------|-----------------------------------------|--------------|----------------------------|
| POST   | `/auth/login`                           | Public       | Get JWT token              |
| POST   | `/auth/register`                        | Public       | Create user                |
| GET    | `/system/status`                        | Any role     | Runtime info + version     |
| GET    | `/users`                                | Super Admin  | List users                 |
| PUT    | `/users/{id}`                           | Super Admin  | Update user                |
| DELETE | `/users/{id}`                           | Super Admin  | Delete user                |
| GET    | `/packages?registry=npm&status=pending` | Consumer+    | List packages              |
| POST   | `/packages/{id}/approve`                | Reg. Admin+  | Approve + trigger dep sync |
| POST   | `/packages/{id}/block`                  | Reg. Admin+  | Block package              |
| GET    | `/audit`                                | Reg. Admin+  | Audit log entries          |
| GET    | `/audit/verify`                         | Reg. Admin+  | Verify hash chain          |
| GET    | `/settings`                             | Reg. Admin+  | List runtime settings      |
| PUT    | `/settings/{key}`                       | Super Admin  | Update a setting           |
| GET    | `/registries`                           | Consumer+    | List registries            |
| PUT    | `/registries/{ecosystem}`               | Super Admin  | Update registry config     |
| GET    | `/policies`                             | Consumer+    | List policies              |
| PUT    | `/policies/{name}`                      | Super Admin  | Update policy              |
| PUT    | `/policies/{name}/toggle`               | Super Admin  | Enable/disable policy      |
| GET    | `/sync/jobs/{id}`                       | Reg. Admin+  | Sync job status            |

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
make web                    # Build web UI
make test                   # Run tests with race detector
make lint                   # golangci-lint
make fmt                    # Format code
make generate               # Run go generate (sqlc etc.)
make docker-up              # Build and start Docker stack
make docker-down            # Stop Docker stack
make docker-rebuild         # Full rebuild without cache
make docker-logs            # Show kantar container logs
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
internal/sync/       Recursive dependency sync engine
internal/config/     TOML config loader with env interpolation
pkg/registry/        Public RegistryPlugin interface
web/                 React 19 + Vite 6 + Tailwind 4 SPA
web/src/i18n/        Translations (en, tr, de)
migrations/          Embedded PostgreSQL migrations
```

## Deployment

### Docker Compose

```bash
make docker-up
```

Uses PostgreSQL with bind-mount volumes at `./docker-data/`. Config file `kantar.toml` is mounted as `/etc/kantar/kantar.toml`.

### Binary

Download from [Releases](https://github.com/KilimcininKorOglu/kantar/releases). Available for Linux, macOS, and Windows (amd64/arm64).

```bash
kantar init              # Generate default config
kantar serve --config kantar.toml
```

## License

This project is licensed under the [MIT License](LICENSE).
