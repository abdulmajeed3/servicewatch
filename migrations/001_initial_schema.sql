-- Migration 001: Initial schema
-- Run: psql $DATABASE_URL -f migrations/001_initial_schema.sql

BEGIN;

-- ── Users ─────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id          BIGSERIAL PRIMARY KEY,
    email       TEXT        NOT NULL UNIQUE,
    name        TEXT        NOT NULL,
    password_hash TEXT      NOT NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- ── Monitors ──────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS monitors (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name             TEXT        NOT NULL,
    url              TEXT        NOT NULL,
    -- check_interval stored in seconds
    check_interval   INTEGER     NOT NULL DEFAULT 300,
    -- request_timeout stored in seconds
    request_timeout  INTEGER     NOT NULL DEFAULT 30,
    -- expected_status: 0 means "any 2xx"
    expected_status  INTEGER     NOT NULL DEFAULT 0,
    status           TEXT        NOT NULL DEFAULT 'pending'
                                 CHECK (status IN ('up','down','pending','paused')),
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    -- next_check_at: when the scheduler should pick this monitor up next
    next_check_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_checked_at  TIMESTAMPTZ,
    last_status_code INTEGER,
    last_response_ms INTEGER,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_monitors_user_id     ON monitors(user_id);
CREATE INDEX IF NOT EXISTS idx_monitors_next_check  ON monitors(next_check_at) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_monitors_status      ON monitors(user_id, status);

-- ── Monitor Checks ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS monitor_checks (
    id           BIGSERIAL   PRIMARY KEY,
    monitor_id   BIGINT      NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    user_id      BIGINT      NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    checked_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_success   BOOLEAN     NOT NULL,
    status_code  INTEGER,
    response_ms  INTEGER,
    -- failure_reason: short description of failure (DNS error, timeout, unexpected status, etc.)
    failure_reason TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_checks_monitor_id   ON monitor_checks(monitor_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_checks_user_id      ON monitor_checks(user_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_checks_recent       ON monitor_checks(monitor_id, checked_at DESC)
    WHERE checked_at > NOW() - INTERVAL '30 days';

-- ── Incidents ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS incidents (
    id           BIGSERIAL   PRIMARY KEY,
    monitor_id   BIGINT      NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    user_id      BIGINT      NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at  TIMESTAMPTZ,
    -- check counts during incident
    failure_count INTEGER     NOT NULL DEFAULT 1,
    -- last_failure_at for sorting/display
    last_failure_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_incidents_monitor_id ON incidents(monitor_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_open       ON incidents(monitor_id) WHERE resolved_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_incidents_user_id    ON incidents(user_id, started_at DESC);

-- ── Refresh Tokens ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          BIGSERIAL   PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- ── Schema Migrations Tracking ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     TEXT        PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO schema_migrations(version) VALUES ('001') ON CONFLICT DO NOTHING;

COMMIT;
