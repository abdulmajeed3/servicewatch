# Deployment

> **Reference only:** this guide documents how the system is designed to run. Deploying or operating this software requires prior written authorization under the [LICENSE](../LICENSE).

This guide covers running ServiceWatch in a production-like environment. ServiceWatch is a single Go binary serving a REST API, with a separate Vite-built static frontend.

---

## Prerequisites

- Linux/macOS server (or Windows with equivalent tooling)
- PostgreSQL 15 or higher
- Go 1.23+ (to compile) or a pre-built binary
- Node.js 22+ (to build the frontend)
- A reverse proxy (nginx, Caddy, or similar) for TLS termination

---

## Build

### Backend

```bash
# Build a static binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/servicewatch ./cmd/server
```

### Frontend

```bash
cd frontend
npm ci --legacy-peer-deps
npm run build
# Output: frontend/dist/
```

---

## Database Setup

```bash
# Create the database user and database
psql -U postgres -c "CREATE USER servicewatch WITH PASSWORD 'your_secure_password';"
psql -U postgres -c "CREATE DATABASE servicewatch OWNER servicewatch;"

# Run migrations
export DATABASE_URL="host=localhost port=5432 user=servicewatch password=your_secure_password dbname=servicewatch sslmode=require"
psql "$DATABASE_URL" -f migrations/001_initial_schema.sql
```

Do not run `002_seed_data.sql` in production — it creates a demo account with a known password.

---

## Environment Variables

Create `/etc/servicewatch/servicewatch.env` (readable only by the service user):

```bash
# Required
DATABASE_URL=host=localhost port=5432 user=servicewatch password=CHANGE_ME dbname=servicewatch sslmode=require
JWT_SECRET=CHANGE_ME_use_a_random_string_of_at_least_32_characters

# Recommended production values
PORT=8080
JWT_EXPIRY=24h
BCRYPT_COST=12
MONITOR_WORKER_COUNT=10
MONITOR_SCHEDULER_INTERVAL=30s
MONITOR_ALLOW_PRIVATE_IPS=false

# CORS: set to your actual frontend domain
CORS_ORIGINS=https://servicewatch.example.com
```

Generate a strong JWT secret:

```bash
openssl rand -base64 48
```

---

## systemd Service

Create `/etc/systemd/system/servicewatch.service`:

```ini
[Unit]
Description=ServiceWatch monitoring service
After=network.target postgresql.service

[Service]
Type=simple
User=servicewatch
Group=servicewatch
WorkingDirectory=/opt/servicewatch
EnvironmentFile=/etc/servicewatch/servicewatch.env
ExecStart=/opt/servicewatch/bin/servicewatch
Restart=on-failure
RestartSec=5s

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/servicewatch

[Install]
WantedBy=multi-user.target
```

```bash
# Create user and deploy
useradd -r -s /bin/false servicewatch
mkdir -p /opt/servicewatch/bin
cp bin/servicewatch /opt/servicewatch/bin/
chmod +x /opt/servicewatch/bin/servicewatch

# Enable and start
systemctl daemon-reload
systemctl enable servicewatch
systemctl start servicewatch
systemctl status servicewatch
```

---

## Nginx Configuration

ServiceWatch's API is served from `/api/v1`; the React SPA is served from `/`. A single nginx server block handles both.

```nginx
server {
    listen 443 ssl http2;
    server_name servicewatch.example.com;

    ssl_certificate     /etc/letsencrypt/live/servicewatch.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/servicewatch.example.com/privkey.pem;

    # Serve the React SPA
    root /var/www/servicewatch;
    index index.html;

    # SPA fallback — all non-file, non-API paths serve index.html
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Proxy API requests to the Go backend
    location /api/ {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
    }

    # Proxy health endpoints
    location /health/ {
        proxy_pass http://127.0.0.1:8080;
    }
}

server {
    listen 80;
    server_name servicewatch.example.com;
    return 301 https://$host$request_uri;
}
```

Deploy the frontend:

```bash
cp -r frontend/dist/* /var/www/servicewatch/
```

---

## Caddy Configuration (alternative)

```caddyfile
servicewatch.example.com {
    root * /var/www/servicewatch
    encode gzip

    handle /api/* {
        reverse_proxy localhost:8080
    }

    handle /health/* {
        reverse_proxy localhost:8080
    }

    handle {
        try_files {path} /index.html
        file_server
    }
}
```

---

## TLS

Use [Let's Encrypt](https://letsencrypt.org/) via Certbot or Caddy's built-in ACME support. The Go server itself does not terminate TLS; the reverse proxy handles it.

---

## Security Checklist

- [ ] `JWT_SECRET` is ≥32 random characters, not reused from `.env.example`
- [ ] `DB_PASSWORD` is a strong random password
- [ ] `MONITOR_ALLOW_PRIVATE_IPS` is `false` (the default)
- [ ] PostgreSQL is not exposed to the public internet (listens on `127.0.0.1` or a private network)
- [ ] The servicewatch process runs as a dedicated non-root user
- [ ] TLS is enabled on the reverse proxy
- [ ] `CORS_ORIGINS` is set to your actual frontend domain, not `*`
- [ ] Firewall rules prevent the monitoring process from reaching your internal services (defense in depth against SSRF)
- [ ] Backups are configured for the PostgreSQL database

---

## Monitoring ServiceWatch Itself

Use another monitoring service (or a second ServiceWatch instance) to check:

- `GET https://servicewatch.example.com/health/ready` — database connectivity
- `GET https://servicewatch.example.com/health/live` — process liveness

---

## Updating

```bash
# Build new binary
go build -ldflags="-s -w" -o bin/servicewatch ./cmd/server

# Deploy
cp bin/servicewatch /opt/servicewatch/bin/servicewatch

# Apply any new migrations
psql "$DATABASE_URL" -f migrations/NNN_description.sql

# Restart
systemctl restart servicewatch
```

New migrations are only added for schema changes; re-running old migrations is safe (they use `ON CONFLICT DO NOTHING` on `schema_migrations`).

---

## Log Output

ServiceWatch writes structured JSON logs to stdout. Redirect them with systemd journal or pipe to a log aggregator:

```bash
journalctl -u servicewatch -f
```

Example log line:

```json
{"time":"2026-10-29T12:00:00Z","level":"INFO","msg":"check complete","monitor_id":"abc...","status":"up","response_ms":142}
```
