-- name: CreateSyncJob :one
INSERT INTO sync_jobs (registry_type, package_name, status)
VALUES ($1, $2, 'pending')
RETURNING id, registry_type, package_name, status, started_at, completed_at, error_message, packages_synced, created_at;

-- name: UpdateSyncJobStatus :exec
UPDATE sync_jobs
SET status          = $1,
    packages_synced = $2,
    error_message   = $3,
    started_at      = CASE WHEN $1 = 'running' AND started_at IS NULL
                           THEN CURRENT_TIMESTAMP ELSE started_at END,
    completed_at    = CASE WHEN $1 IN ('done', 'failed')
                           THEN CURRENT_TIMESTAMP ELSE completed_at END
WHERE id = $4;

-- name: GetSyncJob :one
SELECT id, registry_type, package_name, status, started_at, completed_at, error_message, packages_synced, created_at
FROM sync_jobs WHERE id = $1 LIMIT 1;

-- name: ListSyncJobs :many
SELECT id, registry_type, package_name, status, started_at, completed_at, error_message, packages_synced, created_at
FROM sync_jobs ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: MarkStaleSyncJobsFailed :exec
UPDATE sync_jobs SET status = 'failed', error_message = 'interrupted by server restart'
WHERE status = 'running';
