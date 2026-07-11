SET search_path TO spatial_memory, public, extensions;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS sms_codes;
DROP TABLE IF EXISTS moderation_queue;
DROP TABLE IF EXISTS memory_interactions;
DROP TABLE IF EXISTS memory_permissions;
DROP TABLE IF EXISTS circle_members;
DROP TABLE IF EXISTS friend_circles;
DROP TABLE IF EXISTS memory_media;
DROP TABLE IF EXISTS memories;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS trigger_set_updated_at();
DROP SCHEMA IF EXISTS spatial_memory;

-- PostGIS is shared infrastructure in Supabase. Never drop it from an app rollback.
