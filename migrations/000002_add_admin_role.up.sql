-- Migration: Add admin role column to users table
-- Note: is_admin already exists in the initial schema, this migration is for completeness
-- if the column needs to be added in a fresh migration context

-- Ensure is_admin column exists with proper default
ALTER TABLE users ALTER COLUMN is_admin SET DEFAULT FALSE;

-- Create index for admin lookups
CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users (is_admin) WHERE is_admin = TRUE;
