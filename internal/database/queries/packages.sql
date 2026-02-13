-- name: GetPackage :one
SELECT * FROM packages WHERE registry_type = ? AND name = ? LIMIT 1;

-- name: GetPackageByID :one
SELECT * FROM packages WHERE id = ? LIMIT 1;

-- name: ListPackages :many
SELECT * FROM packages WHERE registry_type = ? ORDER BY name LIMIT ? OFFSET ?;

-- name: ListPackagesByStatus :many
SELECT * FROM packages WHERE registry_type = ? AND status = ? ORDER BY name LIMIT ? OFFSET ?;

-- name: SearchPackages :many
SELECT * FROM packages WHERE registry_type = ? AND name LIKE ? ORDER BY name LIMIT ? OFFSET ?;

-- name: CreatePackage :one
INSERT INTO packages (registry_type, name, description, license, homepage, repository, status, requested_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdatePackageStatus :exec
UPDATE packages SET status = ?, approved_by = ?, blocked_reason = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeletePackage :exec
DELETE FROM packages WHERE id = ?;

-- name: CountPackages :one
SELECT COUNT(*) FROM packages WHERE registry_type = ?;

-- name: CountPackagesByStatus :one
SELECT COUNT(*) FROM packages WHERE registry_type = ? AND status = ?;
