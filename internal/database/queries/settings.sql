-- name: GetSetting :one
SELECT key, value, category, description, updated_at
FROM settings WHERE key = $1 LIMIT 1;

-- name: GetSettingsByCategory :many
SELECT key, value, category, description, updated_at
FROM settings WHERE category = $1 ORDER BY key;

-- name: ListSettings :many
SELECT key, value, category, description, updated_at
FROM settings ORDER BY category, key;

-- name: UpsertSetting :exec
INSERT INTO settings (key, value, category, description, updated_at)
VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = CURRENT_TIMESTAMP;

-- name: UpdateSettingValue :exec
UPDATE settings SET value = $1, updated_at = CURRENT_TIMESTAMP
WHERE key = $2;
