-- name: GetPackageVersion :one
SELECT * FROM package_versions WHERE package_id = ? AND version = ? LIMIT 1;

-- name: ListPackageVersions :many
SELECT * FROM package_versions WHERE package_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CreatePackageVersion :one
INSERT INTO package_versions (package_id, version, size, checksum_sha256, checksum_sha1, storage_path, metadata_json, synced_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeletePackageVersion :exec
DELETE FROM package_versions WHERE id = ?;

-- name: CountPackageVersions :one
SELECT COUNT(*) FROM package_versions WHERE package_id = ?;
