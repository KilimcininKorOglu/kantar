-- Runtime settings table (key-value store for dynamic configuration)
CREATE TABLE IF NOT EXISTS settings (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    category    TEXT NOT NULL DEFAULT 'general',
    description TEXT NOT NULL DEFAULT '',
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
