-- name: CreateAuditLog :one
INSERT INTO audit_logs (event, actor_username, actor_ip, actor_user_agent, resource_registry, resource_package, resource_version, result, metadata_json, prev_hash, hash)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs ORDER BY timestamp DESC LIMIT ? OFFSET ?;

-- name: ListAuditLogsByEvent :many
SELECT * FROM audit_logs WHERE event = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?;

-- name: ListAuditLogsByActor :many
SELECT * FROM audit_logs WHERE actor_username = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?;

-- name: GetLatestAuditLog :one
SELECT * FROM audit_logs ORDER BY id DESC LIMIT 1;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs;
