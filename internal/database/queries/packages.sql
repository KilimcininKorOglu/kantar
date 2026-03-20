-- name: GetPackage :one
SELECT * FROM packages WHERE registry_type = $1 AND name = $2 LIMIT 1;

-- name: GetPackageByID :one
SELECT * FROM packages WHERE id = $1 LIMIT 1;

-- name: ListPackages :many
SELECT * FROM packages WHERE registry_type = $1 ORDER BY name LIMIT $2 OFFSET $3;

-- name: ListPackagesByStatus :many
SELECT * FROM packages WHERE registry_type = $1 AND status = $2 ORDER BY name LIMIT $3 OFFSET $4;

-- name: SearchPackages :many
SELECT * FROM packages WHERE registry_type = $1 AND name LIKE $2 ORDER BY name LIMIT $3 OFFSET $4;

-- name: CreatePackage :one
INSERT INTO packages (registry_type, name, description, license, homepage, repository, status, requested_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdatePackageStatus :exec
UPDATE packages SET status = $1, approved_by = $2, blocked_reason = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $4;

-- name: DeletePackage :exec
DELETE FROM packages WHERE id = $1;

-- name: CountPackages :one
SELECT COUNT(*) FROM packages WHERE registry_type = $1;

-- name: CountPackagesByStatus :one
SELECT COUNT(*) FROM packages WHERE registry_type = $1 AND status = $2;
