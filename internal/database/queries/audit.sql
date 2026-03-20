-- name: CreateAuditLog :one
INSERT INTO audit_logs (event, actor_username, actor_ip, actor_user_agent, resource_registry, resource_package, resource_version, result, metadata_json, prev_hash, hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs ORDER BY timestamp DESC LIMIT $1 OFFSET $2;

-- name: ListAuditLogsByEvent :many
SELECT * FROM audit_logs WHERE event = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByActor :many
SELECT * FROM audit_logs WHERE actor_username = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3;

-- name: GetLatestAuditLog :one
SELECT * FROM audit_logs ORDER BY id DESC LIMIT 1;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs;
