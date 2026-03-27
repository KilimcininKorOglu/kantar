-- name: ListPolicies :many
SELECT id, name, policy_type, config_toml, enabled, created_at, updated_at
FROM policies ORDER BY name;

-- name: GetPolicy :one
SELECT id, name, policy_type, config_toml, enabled, created_at, updated_at
FROM policies WHERE name = $1 LIMIT 1;

-- name: UpsertPolicy :exec
INSERT INTO policies (name, policy_type, config_toml, enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (name) DO UPDATE SET
    config_toml = $3,
    enabled = $4,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpdatePolicyEnabled :exec
UPDATE policies SET enabled = $1, updated_at = CURRENT_TIMESTAMP
WHERE name = $2;
