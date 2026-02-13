-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ? LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY username LIMIT ? OFFSET ?;

-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, role)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateUser :exec
UPDATE users SET email = ?, role = ?, active = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;
