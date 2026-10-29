# Security

This document describes the security design of ServiceWatch, the threat model for outbound HTTP monitoring, and known limitations.

---

## SSRF (Server-Side Request Forgery)

ServiceWatch makes outbound HTTP requests to URLs supplied by authenticated users. This creates a Server-Side Request Forgery risk: a malicious user could supply a URL that targets internal services, cloud metadata endpoints, or loopback addresses.

### Defense-in-Depth

SSRF is mitigated at two independent layers.

#### Layer 1 — URL Validation at Creation Time

`internal/monitor/validate.go` rejects URLs that are obviously unsafe before a monitor is created or updated:

| Check | What is rejected |
|---|---|
| Scheme | Anything other than `http` or `https` |
| Embedded credentials | `http://user:pass@example.com` — `url.User != nil` |
| Obvious localhost | `localhost`, `127.x.x.x`, `::1` (checked via `net.ParseIP().IsLoopback()`) |
| Link-local unicast | `169.254.x.x`, `fe80::…` (catches AWS metadata IP) |
| URL fragments | `#anchor` in a monitor URL is rejected |
| Malformed URLs | `url.Parse` error or empty host |

This layer is fast and catches the most obvious attacks, but it cannot prevent all SSRF because DNS is not resolved at validation time.

#### Layer 2 — IP Validation at Dial Time

`internal/checker/transport.go` implements a custom `ssrfDialer` that intercepts every outbound connection:

```go
func (d *ssrfDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
    host, port, _ := net.SplitHostPort(addr)
    addrs, err := net.DefaultResolver.LookupHost(ctx, host)
    if err != nil {
        return nil, err
    }
    if !d.allowPrivateIPs {
        for _, raw := range addrs {
            ip := net.ParseIP(raw)
            if isBlockedIP(ip) {
                return nil, fmt.Errorf("%w: %s resolves to %s", ErrBlockedAddress, host, ip)
            }
        }
    }
    // Connect only to the first validated address.
    target := net.JoinHostPort(addrs[0], port)
    return d.inner.DialContext(ctx, network, target)
}
```

`isBlockedIP` blocks:

| Class | Example |
|---|---|
| Loopback | `127.0.0.1`, `::1` |
| Private (RFC 1918) | `10.x`, `172.16–31.x`, `192.168.x` |
| Link-local | `169.254.x.x` (AWS metadata at `169.254.169.254`) |
| IPv6 link-local | `fe80::/10` |
| Multicast | `224.x.x.x` |
| Unspecified | `0.0.0.0`, `::` |

The transport also:
- Sets `Proxy: nil` (ignores `HTTP_PROXY`/`HTTPS_PROXY` environment variables that could route traffic through attacker-controlled proxies).
- Sets `DisableKeepAlives: true` to prevent connection reuse across checks.
- Limits the response body read to 1 MiB via `io.LimitReader`.
- Disables redirects — `CheckRedirect` returns `http.ErrUseLastResponse`. Following redirects would require re-validating each destination; disabling them is simpler and safer.

### Development Override

`MONITOR_ALLOW_PRIVATE_IPS=true` disables Layer 2 IP validation. This exists so test servers on loopback can be checked during development.

**Never set `MONITOR_ALLOW_PRIVATE_IPS=true` in production.**

### Known Limitations

#### DNS Rebinding

The most significant remaining risk is DNS rebinding:

1. Attacker registers `evil.example.com` with a short TTL.
2. During URL validation (Layer 1), the domain resolves to a public IP — passes.
3. Attacker changes the DNS record to `192.168.1.1`.
4. When the check fires, `ssrfDialer` resolves the domain and sees the private IP — blocked.

However, there is a window between Layer 2's DNS lookup and the TCP connection where a rebinding attack could in theory succeed if the OS or Go resolver caches the public IP and a race condition occurs. This window is very small but non-zero.

**Recommended mitigation in production:** Add network-level egress controls (firewall rules, security groups) that prevent the monitoring process from reaching your internal network, regardless of what DNS returns.

#### No Mutual TLS Verification of Monitored Endpoints

ServiceWatch verifies TLS certificates of monitored endpoints using the system certificate pool. A self-signed certificate or expired certificate will cause the check to fail. There is intentionally no option to skip TLS verification (`InsecureSkipVerify`) — this would defeat the purpose of monitoring certificate health.

---

## Authentication

### JWT

- Algorithm: HS256 (HMAC-SHA256)
- Secret: `JWT_SECRET` environment variable, minimum 32 characters
- Expiry: configurable via `JWT_EXPIRY` (default 24 hours)
- Tokens are stateless; there is no server-side revocation in this version

### Password Hashing

- Algorithm: bcrypt
- Default cost: 12 (configurable via `BCRYPT_COST`)
- Minimum cost enforced: 10 (below this bcrypt is too fast)
- Password strength requirements: 8–128 characters, at least one uppercase letter, one lowercase letter, one digit

### Email Enumeration Protection

`auth.Service.Login` returns the same error (`ErrInvalidCredentials`) whether the email is not found or the password is wrong. This prevents attackers from enumerating valid email addresses.

### Ownership Isolation

Every authenticated API handler reads the `userID` from the JWT claims (attached to the request context by the auth middleware) and filters all database queries by that user ID. A user cannot read or modify another user's monitors.

---

## Input Validation

- Monitor URLs: validated by `monitor.ValidateCreate` and `monitor.ValidateUpdate` before storage
- Intervals: must be one of `{60, 300, 600, 1800, 3600}` seconds
- Timeouts: must be 5–120 seconds
- Expected status: must be 0 (any 2xx) or a valid HTTP status code 100–599
- All user-supplied strings are stored as-is and returned as JSON; SQL injection is not possible because all queries use parameterized statements via pgx

---

## API Security Headers

The HTTP server sets:

| Header | Value |
|---|---|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Content-Security-Policy` | `default-src 'self'` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |

CORS is configured via `CORS_ORIGINS` (comma-separated list). Only those origins receive `Access-Control-Allow-Origin`.

---

## Sensitive Data

- JWT secrets and database passwords must be set in the environment, never committed to source control.
- `.env` is in `.gitignore`.
- Seed data (`002_seed_data.sql`) uses a pre-computed bcrypt hash. The plaintext `Demo1234!` is documented in that file's comment only — it is not stored anywhere in the application code.
- Passwords are never logged.
- JWT tokens are logged only in debug mode (not the default).

---

## Dependency Supply Chain

Dependencies are pinned to exact versions in `go.mod` and `go.sum`. The `go mod verify` step in CI checks that module checksums match `go.sum`. Frontend dependencies are locked via `package-lock.json`.

---

## Reporting Vulnerabilities

This is a portfolio project. If you find a security issue, please open a GitHub issue. For a sensitive finding, email the address in the project's GitHub profile.
