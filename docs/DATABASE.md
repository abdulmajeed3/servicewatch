# Database

ServiceWatch uses PostgreSQL (version 15 or higher). Schema is managed through explicit SQL migration files in `migrations/`.

---

## Running Migrations

Migrations are plain SQL files intended to be run in order with `psql`. There is no migration runner binary.

```bash
psql "$DATABASE_URL" -f migrations/001_initial_schema.sql
psql "$DATABASE_URL" -f migrations/002_seed_data.sql   # optional dev seed
```

Migration order is tracked by the `schema_migrations` table:

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     TEXT PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Each migration file inserts its version at the end (e.g., `INSERT INTO schema_migrations (version) VALUES ('001') ON CONFLICT DO NOTHING`), so re-running a migration is safe.

---

## Schema

### `users`

```sql
CREATE TABLE users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT         NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    display_name  TEXT         NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

Email has a `UNIQUE` constraint. The auth store returns a domain error (`ErrEmailTaken`) when a `23505` unique-violation is returned by PostgreSQL.

---

### `monitors`

```sql
CREATE TABLE monitors (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                 TEXT         NOT NULL,
    url                  TEXT         NOT NULL,
    interval_seconds     INTEGER      NOT NULL DEFAULT 300,
    timeout_seconds      INTEGER      NOT NULL DEFAULT 30,
    expected_status      INTEGER      NOT NULL DEFAULT 0,
    is_active            BOOLEAN      NOT NULL DEFAULT TRUE,
    status               TEXT         NOT NULL DEFAULT 'pending',
    consecutive_failures INTEGER      NOT NULL DEFAULT 0,
    last_checked_at      TIMESTAMPTZ,
    next_check_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_response_ms     INTEGER,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitors_next_check_at ON monitors (next_check_at)
    WHERE is_active = TRUE;
```

The partial index on `next_check_at WHERE is_active = TRUE` makes the scheduler's polling query (`WHERE is_active = TRUE AND next_check_at <= NOW()`) an index scan rather than a sequential scan, even as the table grows.

**`status`** values: `pending`, `up`, `down`, `paused`.

**`expected_status = 0`** means "accept any 2xx status code". Any other value means the response must match exactly.

**`next_check_at`** is updated after each check to `NOW() + interval_seconds * interval '1 second'`.

---

### `monitor_checks`

```sql
CREATE TABLE monitor_checks (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id    UUID         NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    checked_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    is_success    BOOLEAN      NOT NULL,
    status_code   INTEGER,
    response_ms   INTEGER,
    error_message TEXT
);

CREATE INDEX idx_monitor_checks_monitor_id_checked_at
    ON monitor_checks (monitor_id, checked_at DESC);
```

`error_message` is set on failed checks and contains the Go error string (e.g., `context deadline exceeded`, `connection refused`, `SSRF: host resolves to blocked address`).

---

### `incidents`

```sql
CREATE TABLE incidents (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id       UUID         NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    started_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    resolved_at      TIMESTAMPTZ,
    failure_count    INTEGER      NOT NULL DEFAULT 1,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incidents_monitor_id_resolved_at
    ON incidents (monitor_id, resolved_at);
```

**Incident lifecycle:**

- One open incident per monitor at most (enforced in `incident.Store.OpenOrIncrement`).
- On each failed check: if no open incident exists, `INSERT` a new one; if one exists, `UPDATE failure_count = failure_count + 1`.
- On first successful check: `UPDATE resolved_at = NOW()` on the open incident.

---

### `refresh_tokens`

```sql
CREATE TABLE refresh_tokens (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT         NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

Present in schema for future refresh-token support. Not used in the current implementation.

---

## Relationships

```
users ──< monitors ──< monitor_checks
                  ──< incidents
       ──< refresh_tokens
```

All child tables use `ON DELETE CASCADE` so deleting a user removes all their monitors, checks, and incidents.

---

## Availability Calculation

```sql
SELECT
    COUNT(*) FILTER (WHERE is_success)::FLOAT / NULLIF(COUNT(*), 0) * 100
FROM monitor_checks
WHERE monitor_id = $1
  AND checked_at >= NOW() - $2::INTERVAL;
```

If there are no checks in the window, the result is `NULL` (reported as `0%` in the API).

---

## Response Time Series

```sql
SELECT checked_at, response_ms
FROM monitor_checks
WHERE monitor_id = $1
  AND checked_at >= NOW() - $2::INTERVAL
ORDER BY checked_at ASC;
```

`response_ms` is `NULL` for failed checks where no response was received. The frontend filters these out when rendering the chart.

---

## History Retention

Check history is retained indefinitely by default. For long-running deployments, consider a scheduled pruning job:

```sql
-- Delete checks older than 90 days
DELETE FROM monitor_checks
WHERE checked_at < NOW() - INTERVAL '90 days';
```

Row estimates at steady state:

| Interval | Checks/day/monitor | Rows at 90 days (100 monitors) |
|---|---|---|
| 1 min | 1,440 | ~12.9M |
| 5 min | 288 | ~2.6M |
| 30 min | 48 | ~432K |

At 5-minute intervals with 20 monitors, 90-day retention produces approximately 500K rows — well within PostgreSQL's comfortable range.

---

## Connection Pool

The application uses `pgxpool` (pgx's built-in pool):

```go
pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
```

Default pool settings are used (max 4 connections per CPU core). For production, consider tuning `pool_max_conns` in the database URL:

```
postgresql://user:pass@host/db?pool_max_conns=20
```

---

## Backup

For production, use `pg_dump`:

```bash
pg_dump -Fc "$DATABASE_URL" > servicewatch_$(date +%Y%m%d).dump
```

Restore with:

```bash
pg_restore -d "$DATABASE_URL" servicewatch_20260101.dump
```
