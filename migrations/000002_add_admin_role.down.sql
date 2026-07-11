-- Migration: Remove admin role additions

SET search_path TO spatial_memory, public, extensions;

DROP INDEX IF EXISTS idx_users_is_admin;

-- Note: We don't remove the is_admin column as it's part of the base schema
-- and removing it could break existing code
