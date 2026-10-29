# Troubleshooting

---

## Server Won't Start

### `JWT_SECRET must be set`

The `JWT_SECRET` environment variable is missing or empty.

**Fix:** Set it in `.env` or your environment. It must be at least 32 characters.

```bash
export JWT_SECRET="$(openssl rand -base64 48)"
```

### `failed to connect to database`

The server could not reach PostgreSQL.

**Check:**
1. PostgreSQL is running: `pg_isready -h localhost -p 5432`
2. `DATABASE_URL` is set correctly
3. The database user exists and has the right permissions
4. `sslmode` is appropriate for your setup (`disable` for local, `require` for production)

### `address already in use`

Something is already listening on port 8080.

**Fix:**
```bash
# Find the process
lsof -i :8080   # Linux/macOS

# Or change the port
export PORT=8081
```

---

## Monitor Creation Fails

### `URL contains embedded credentials`

The URL includes a username or password (`http://user:pass@host`).

**Fix:** Remove the credentials from the URL.

### `URL scheme must be http or https`

The URL starts with `ftp://`, `file://`, or another unsupported scheme.

**Fix:** Use an `http://` or `https://` URL.

### `URL targets a local or private address`

The URL resolves to a loopback or private IP. This is the SSRF protection in action.

**Fix (development only):** Set `MONITOR_ALLOW_PRIVATE_IPS=true`. Never set this in production.

### `interval_seconds must be one of…`

The requested interval is not in the allowed set {60, 300, 600, 1800, 3600}.

**Fix:** Use one of the supported intervals.

---

## Checks Are Failing

### `connection refused`

The target server refused the connection. Either the server is down, the port is wrong, or a firewall is blocking the request.

### `context deadline exceeded`

The check timed out. The target server did not respond within `timeout_seconds`.

**Fix:** Increase `timeout_seconds` (up to 120) or investigate the target server's latency.

### `SSRF: host resolves to a blocked address`

The hostname resolved to a private or loopback IP. This can happen if:
- The target URL was valid at creation time but the DNS record now points to a private IP
- The target is behind a split-horizon DNS setup that returns a private IP for internal callers

**Fix (development only):** Set `MONITOR_ALLOW_PRIVATE_IPS=true`.  
**Fix (production):** The target URL must resolve to a publicly routable IP.

### `x509: certificate signed by unknown authority`

The target server uses a self-signed or internally-signed TLS certificate.

ServiceWatch uses the system certificate pool. Self-signed certs are intentionally rejected — a cert that your machine doesn't trust means a monitor can't verify the endpoint's identity.

**Fix:** Use a publicly trusted TLS certificate on the monitored endpoint (Let's Encrypt is free).

### Monitor is stuck in `pending`

The scheduler has not run a check yet.

**Check:** The scheduler runs every `MONITOR_SCHEDULER_INTERVAL` seconds (default 30s). Wait one interval after creating the monitor. If it stays `pending` longer than 60 seconds, check the server logs for scheduler errors.

---

## Authentication Issues

### `invalid or expired token`

The JWT has expired (`JWT_EXPIRY` default is 24h) or the secret changed.

**Fix:** Log in again to obtain a new token.

### `JWT_SECRET must be at least 32 characters`

The configured secret is too short.

**Fix:** Use a longer secret.

---

## Frontend Issues

### `npm install` fails with peer dependency errors

Use `--legacy-peer-deps`:

```bash
npm install --legacy-peer-deps
```

This is expected due to version constraints between React 19 and some testing libraries.

### API calls return 401 from the browser

1. Check that the backend is running: `curl http://localhost:8080/health/live`
2. Check that the Vite proxy is configured correctly in `vite.config.ts`
3. Check that the token in `localStorage` is valid: open DevTools → Application → Local Storage → `sw_token`

### TypeScript errors in test files

Test files use `@testing-library/jest-dom` types that conflict with the production build's lib settings.

**Fix:** Ensure `tsconfig.app.json` has `"exclude": ["src/test/**"]` and that `tsconfig.test.json` exists. See [docs/TESTING.md](TESTING.md).

---

## Database Issues

### Migration already applied

If you re-run a migration, it will attempt to `INSERT` into `schema_migrations`. Because the `version` column is a `PRIMARY KEY`, the `ON CONFLICT DO NOTHING` clause silently skips the re-insert, and `IF NOT EXISTS` guards on `CREATE TABLE` prevent schema conflicts.

**Re-running a migration is safe.**

### `ERROR: relation "monitors" already exists`

You tried to run `001_initial_schema.sql` a second time without the `IF NOT EXISTS` guards. The current migration uses `CREATE TABLE IF NOT EXISTS` throughout, so this should not happen.

### Disk space growing large

Check row counts:

```sql
SELECT relname, n_live_tup FROM pg_stat_user_tables ORDER BY n_live_tup DESC;
```

`monitor_checks` grows with every check. See [docs/DATABASE.md](DATABASE.md) for pruning guidance.

---

## CI Failures

### `gofmt` check fails

One or more Go files need formatting.

**Fix:**
```bash
gofmt -w ./cmd ./internal
```

### `go test` fails with `-race`

A data race was detected. The race detector is enabled in CI (`-race` flag).

**Investigate:** Run `go test -race -v ./...` locally to see the race report.

### Frontend typecheck fails

```bash
cd frontend && npm run typecheck
```

Review the TypeScript errors. Common causes:
- A type changed in the backend's JSON response shape
- A new component uses a prop that isn't in the type definition
- Test files are included in the production tsconfig

### `vite build` fails with chunk size warning

The bundle exceeds the default size warning threshold (500 KB). This is a **warning**, not an error. The current build produces an ~816 KB bundle and still passes.

To suppress: set `build.chunkSizeWarningLimit` in `vite.config.ts`. To fix properly: add code-splitting with `React.lazy`.
