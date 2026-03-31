-- Add locale column to users table for per-user language preference
ALTER TABLE users ADD COLUMN locale TEXT NOT NULL DEFAULT 'en';
