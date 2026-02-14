-- name: GetAPIToken :one
SELECT * FROM api_tokens WHERE token_hash = ? LIMIT 1;

-- name: ListAPITokensByUser :many
SELECT * FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC;

-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, name, token_hash, token_prefix, expires_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteAPIToken :exec
DELETE FROM api_tokens WHERE id = ?;

-- name: UpdateAPITokenLastUsed :exec
UPDATE api_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: GetUserRole :one
SELECT * FROM user_roles WHERE user_id = ? AND role = ? AND registry_type = ? AND namespace = ? LIMIT 1;

-- name: ListUserRoles :many
SELECT * FROM user_roles WHERE user_id = ?;

-- name: CreateUserRole :one
INSERT INTO user_roles (user_id, role, registry_type, namespace)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: DeleteUserRole :exec
DELETE FROM user_roles WHERE id = ?;
