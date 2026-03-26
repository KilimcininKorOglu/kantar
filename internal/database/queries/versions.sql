-- name: GetPackageVersion :one
SELECT * FROM package_versions WHERE package_id = $1 AND version = $2 LIMIT 1;

-- name: ListPackageVersions :many
SELECT * FROM package_versions WHERE package_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CreatePackageVersion :one
INSERT INTO package_versions (package_id, version, size, checksum_sha256, checksum_sha1, storage_path, metadata_json, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: DeletePackageVersion :exec
DELETE FROM package_versions WHERE id = $1;

-- name: CountPackageVersions :one
SELECT COUNT(*) FROM package_versions WHERE package_id = $1;

-- name: UpsertPackageVersion :one
INSERT INTO package_versions (package_id, version, size, checksum_sha256, checksum_sha1, storage_path, metadata_json, synced_at)
VALUES ($1, $2, 0, '', '', '', '{}', CURRENT_TIMESTAMP)
ON CONFLICT (package_id, version) DO UPDATE SET synced_at = CURRENT_TIMESTAMP
RETURNING *;
