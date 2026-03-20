-- name: GetAPIToken :one
SELECT * FROM api_tokens WHERE token_hash = $1 LIMIT 1;

-- name: ListAPITokensByUser :many
SELECT * FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, name, token_hash, token_prefix, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteAPIToken :exec
DELETE FROM api_tokens WHERE id = $1;

-- name: UpdateAPITokenLastUsed :exec
UPDATE api_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: GetUserRole :one
SELECT * FROM user_roles WHERE user_id = $1 AND role = $2 AND registry_type = $3 AND namespace = $4 LIMIT 1;

-- name: ListUserRoles :many
SELECT * FROM user_roles WHERE user_id = $1;

-- name: CreateUserRole :one
INSERT INTO user_roles (user_id, role, registry_type, namespace)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteUserRole :exec
DELETE FROM user_roles WHERE id = $1;
