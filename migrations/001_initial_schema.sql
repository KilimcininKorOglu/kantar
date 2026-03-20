-- Kantar initial schema
-- PostgreSQL schema

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'consumer',
    active BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_roles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    registry_type TEXT NOT NULL DEFAULT '*',
    namespace TEXT NOT NULL DEFAULT '*',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, role, registry_type, namespace)
);

CREATE TABLE IF NOT EXISTS api_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    token_prefix TEXT NOT NULL,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS registries (
    id BIGSERIAL PRIMARY KEY,
    ecosystem TEXT NOT NULL UNIQUE,
    mode TEXT NOT NULL DEFAULT 'allowlist',
    upstream TEXT NOT NULL DEFAULT '',
    auto_sync BIGINT NOT NULL DEFAULT 0,
    auto_sync_interval TEXT NOT NULL DEFAULT '6h',
    max_versions BIGINT NOT NULL DEFAULT 0,
    enabled BIGINT NOT NULL DEFAULT 1,
    config_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS packages (
    id BIGSERIAL PRIMARY KEY,
    registry_type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    license TEXT NOT NULL DEFAULT '',
    homepage TEXT NOT NULL DEFAULT '',
    repository TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    requested_by TEXT NOT NULL DEFAULT '',
    approved_by TEXT NOT NULL DEFAULT '',
    blocked_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(registry_type, name)
);

CREATE INDEX IF NOT EXISTS idx_packages_registry_type ON packages(registry_type);
CREATE INDEX IF NOT EXISTS idx_packages_status ON packages(status);
CREATE INDEX IF NOT EXISTS idx_packages_name ON packages(name);

CREATE TABLE IF NOT EXISTS package_versions (
    id BIGSERIAL PRIMARY KEY,
    package_id BIGINT NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    checksum_sha256 TEXT NOT NULL DEFAULT '',
    checksum_sha1 TEXT NOT NULL DEFAULT '',
    storage_path TEXT NOT NULL DEFAULT '',
    deprecated BIGINT NOT NULL DEFAULT 0,
    yanked BIGINT NOT NULL DEFAULT 0,
    metadata_json TEXT NOT NULL DEFAULT '{}',
    synced_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(package_id, version)
);

CREATE INDEX IF NOT EXISTS idx_package_versions_package_id ON package_versions(package_id);

CREATE TABLE IF NOT EXISTS package_dependencies (
    id BIGSERIAL PRIMARY KEY,
    version_id BIGINT NOT NULL REFERENCES package_versions(id) ON DELETE CASCADE,
    dep_name TEXT NOT NULL,
    dep_version_range TEXT NOT NULL DEFAULT '*',
    dep_optional BIGINT NOT NULL DEFAULT 0,
    dep_dev BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_package_dependencies_version_id ON package_dependencies(version_id);

CREATE TABLE IF NOT EXISTS policies (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    policy_type TEXT NOT NULL,
    config_toml TEXT NOT NULL,
    enabled BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event TEXT NOT NULL,
    actor_username TEXT NOT NULL DEFAULT '',
    actor_ip TEXT NOT NULL DEFAULT '',
    actor_user_agent TEXT NOT NULL DEFAULT '',
    resource_registry TEXT NOT NULL DEFAULT '',
    resource_package TEXT NOT NULL DEFAULT '',
    resource_version TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL DEFAULT 'success',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    prev_hash TEXT NOT NULL DEFAULT '',
    hash TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_logs_event ON audit_logs(event);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor_username);

CREATE TABLE IF NOT EXISTS sync_jobs (
    id BIGSERIAL PRIMARY KEY,
    registry_type TEXT NOT NULL,
    package_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT NOT NULL DEFAULT '',
    packages_synced BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_jobs_status ON sync_jobs(status);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INT PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
