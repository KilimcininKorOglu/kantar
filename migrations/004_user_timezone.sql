-- Add timezone column to users table for per-user display timezone
ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC';
