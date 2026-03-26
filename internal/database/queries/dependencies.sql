-- name: InsertPackageDependency :exec
INSERT INTO package_dependencies (version_id, dep_name, dep_version_range, dep_optional, dep_dev)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (version_id, dep_name) DO NOTHING;

-- name: GetPackageDependencies :many
SELECT id, version_id, dep_name, dep_version_range, dep_optional, dep_dev
FROM package_dependencies
WHERE version_id = $1;
