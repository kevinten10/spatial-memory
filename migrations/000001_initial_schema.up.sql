-- Enable PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;

-- =============================================================================
-- Trigger function: auto-update updated_at timestamp
-- =============================================================================
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Users
-- =============================================================================
CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    phone           VARCHAR(20) UNIQUE,
    wechat_open_id  VARCHAR(128) UNIQUE,
    nickname        VARCHAR(50) NOT NULL DEFAULT '',
    avatar_url      TEXT NOT NULL DEFAULT '',
    bio             TEXT NOT NULL DEFAULT '',
    status          SMALLINT NOT NULL DEFAULT 1,  -- 0=banned, 1=active
    is_admin        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_users_phone ON users (phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_users_wechat_open_id ON users (wechat_open_id) WHERE wechat_open_id IS NOT NULL;

-- =============================================================================
-- Memories (core entity with spatial column)
-- =============================================================================
CREATE TABLE memories (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(200) NOT NULL DEFAULT '',
    content         TEXT NOT NULL DEFAULT '',
    location        GEOGRAPHY(POINT, 4326) NOT NULL,
    address         VARCHAR(500) NOT NULL DEFAULT '',
    visibility      SMALLINT NOT NULL DEFAULT 0,  -- 0=private, 1=circle, 2=public
    status          SMALLINT NOT NULL DEFAULT 1,  -- 0=deleted, 1=active, 2=pending_review, 3=rejected
    like_count      INTEGER NOT NULL DEFAULT 0,
    view_count      INTEGER NOT NULL DEFAULT 0,
    bookmark_count  INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_memories_updated_at
    BEFORE UPDATE ON memories
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Primary spatial index for all nearby queries
CREATE INDEX idx_memories_location ON memories USING GIST (location);

-- Composite partial index for public discovery (the hot path)
CREATE INDEX idx_memories_active_public ON memories (created_at DESC)
    WHERE status = 1 AND visibility = 2;

-- User's own memories
CREATE INDEX idx_memories_user_id ON memories (user_id, created_at DESC);

-- =============================================================================
-- Memory media (photos, videos, voice)
-- =============================================================================
CREATE TABLE memory_media (
    id              BIGSERIAL PRIMARY KEY,
    memory_id       BIGINT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_type      SMALLINT NOT NULL,  -- 0=photo, 1=video, 2=voice
    storage_key     TEXT NOT NULL,
    url             TEXT NOT NULL DEFAULT '',
    content_hash    VARCHAR(64) NOT NULL DEFAULT '',  -- SHA-256 for dedup
    file_size       BIGINT NOT NULL DEFAULT 0,
    mime_type       VARCHAR(100) NOT NULL DEFAULT '',
    duration        INTEGER NOT NULL DEFAULT 0,  -- seconds, for video/voice
    width           INTEGER NOT NULL DEFAULT 0,
    height          INTEGER NOT NULL DEFAULT 0,
    sort_order      SMALLINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_memory_media_memory_id ON memory_media (memory_id, sort_order);
CREATE INDEX idx_memory_media_content_hash ON memory_media (content_hash)
    WHERE content_hash != '';

-- =============================================================================
-- Friend circles
-- =============================================================================
CREATE TABLE friend_circles (
    id              BIGSERIAL PRIMARY KEY,
    owner_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_friend_circles_updated_at
    BEFORE UPDATE ON friend_circles
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_friend_circles_owner_id ON friend_circles (owner_id);

-- =============================================================================
-- Circle members
-- =============================================================================
CREATE TABLE circle_members (
    id              BIGSERIAL PRIMARY KEY,
    circle_id       BIGINT NOT NULL REFERENCES friend_circles(id) ON DELETE CASCADE,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (circle_id, user_id)
);

CREATE INDEX idx_circle_members_user_id ON circle_members (user_id);

-- =============================================================================
-- Memory permissions (grant access to circles, users, or via share tokens)
-- =============================================================================
CREATE TABLE memory_permissions (
    id              BIGSERIAL PRIMARY KEY,
    memory_id       BIGINT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    circle_id       BIGINT REFERENCES friend_circles(id) ON DELETE CASCADE,
    user_id         BIGINT REFERENCES users(id) ON DELETE CASCADE,
    token_hash      VARCHAR(64),  -- SHA-256 of share token
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (circle_id IS NOT NULL)::int +
        (user_id IS NOT NULL)::int +
        (token_hash IS NOT NULL)::int = 1
    )
);

CREATE INDEX idx_memory_permissions_memory_id ON memory_permissions (memory_id);
CREATE INDEX idx_memory_permissions_token_hash ON memory_permissions (token_hash)
    WHERE token_hash IS NOT NULL;

-- =============================================================================
-- Memory interactions (likes, bookmarks, reports)
-- =============================================================================
CREATE TABLE memory_interactions (
    id              BIGSERIAL PRIMARY KEY,
    memory_id       BIGINT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    interaction_type SMALLINT NOT NULL,  -- 0=like, 1=bookmark, 2=report
    report_reason   TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (memory_id, user_id, interaction_type)
);

CREATE INDEX idx_memory_interactions_user_id ON memory_interactions (user_id, interaction_type);

-- =============================================================================
-- Moderation queue
-- =============================================================================
CREATE TABLE moderation_queue (
    id              BIGSERIAL PRIMARY KEY,
    memory_id       BIGINT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    status          SMALLINT NOT NULL DEFAULT 0,  -- 0=pending, 1=approved, 2=rejected, 3=escalated
    ai_safe_score   REAL,
    ai_categories   JSONB,
    reviewer_id     BIGINT REFERENCES users(id),
    review_note     TEXT NOT NULL DEFAULT '',
    report_count    INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at     TIMESTAMPTZ
);

CREATE INDEX idx_moderation_queue_status ON moderation_queue (status, created_at)
    WHERE status IN (0, 3);
CREATE INDEX idx_moderation_queue_memory_id ON moderation_queue (memory_id);

-- =============================================================================
-- SMS verification codes
-- =============================================================================
CREATE TABLE sms_codes (
    id              BIGSERIAL PRIMARY KEY,
    phone           VARCHAR(20) NOT NULL,
    code            VARCHAR(6) NOT NULL,
    used            BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sms_codes_phone_active ON sms_codes (phone, created_at DESC)
    WHERE used = FALSE;

-- =============================================================================
-- Refresh tokens
-- =============================================================================
CREATE TABLE refresh_tokens (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 of raw token
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id)
    WHERE revoked_at IS NULL;
