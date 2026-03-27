-- name: ListRegistries :many
SELECT id, ecosystem, mode, upstream, auto_sync, auto_sync_interval, max_versions, enabled, config_json, created_at, updated_at
FROM registries ORDER BY ecosystem;

-- name: GetRegistry :one
SELECT id, ecosystem, mode, upstream, auto_sync, auto_sync_interval, max_versions, enabled, config_json, created_at, updated_at
FROM registries WHERE ecosystem = $1 LIMIT 1;

-- name: UpsertRegistry :exec
INSERT INTO registries (ecosystem, mode, upstream, auto_sync, auto_sync_interval, max_versions, enabled, config_json)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (ecosystem) DO UPDATE SET
    mode = $2,
    upstream = $3,
    auto_sync = $4,
    auto_sync_interval = $5,
    max_versions = $6,
    enabled = $7,
    config_json = $8,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpdateRegistryEnabled :exec
UPDATE registries SET enabled = $1, updated_at = CURRENT_TIMESTAMP
WHERE ecosystem = $2;
