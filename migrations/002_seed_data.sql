-- Migration 002: Development seed data
-- WARNING: Only run in development environments. Contains no real credentials.
-- Run: psql $DATABASE_URL -f migrations/002_seed_data.sql

BEGIN;

-- Demo user: email=demo@servicewatch.dev  password=Demo1234!
-- Password hash generated with bcrypt cost=12
INSERT INTO users (email, name, password_hash) VALUES (
    'demo@servicewatch.dev',
    'Demo User',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ/Ce2'
) ON CONFLICT (email) DO NOTHING;

-- Example monitors pointing to safe, well-known public endpoints
-- NOTE: These URLs must be safe for the environment. Adjust before use.
INSERT INTO monitors (user_id, name, url, check_interval, request_timeout, expected_status)
SELECT
    u.id,
    m.name,
    m.url,
    m.interval_s,
    m.timeout_s,
    m.expected
FROM
    (SELECT id FROM users WHERE email = 'demo@servicewatch.dev' LIMIT 1) u
    CROSS JOIN (VALUES
        ('GitHub API',         'https://api.github.com',       300, 30, 200),
        ('Cloudflare DNS',     'https://1.1.1.1',              300, 30, 200),
        ('HTTPBin Status 200', 'https://httpbin.org/status/200', 300, 30, 200)
    ) AS m(name, url, interval_s, timeout_s, expected)
ON CONFLICT DO NOTHING;

INSERT INTO schema_migrations(version) VALUES ('002') ON CONFLICT DO NOTHING;

COMMIT;
