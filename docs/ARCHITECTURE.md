# Architecture

ServiceWatch is a monolithic application with a clear internal package boundary between the Go backend and the React frontend. They communicate exclusively through the REST API.

---

## Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  Browser                                                        │
│  React 19 + TypeScript + Vite                                   │
│  TanStack Query · React Router · Tailwind                       │
└───────────────────────┬─────────────────────────────────────────┘
                        │  HTTP  /api/v1
┌───────────────────────▼─────────────────────────────────────────┐
│  HTTP Server  (net/http + httprouter)                           │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Middleware chain                                         │   │
│  │  CORS → Request Logger → Auth (JWT)                      │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌────────────┐  ┌───────────────┐  ┌──────────────────────┐   │
│  │  Auth      │  │  Monitor      │  │  History / Dashboard  │   │
│  │  handlers  │  │  handlers     │  │  handlers            │   │
│  └─────┬──────┘  └──────┬────────┘  └──────────┬───────────┘   │
│        │                │                       │               │
│  ┌─────▼──────┐  ┌──────▼────────┐  ┌──────────▼───────────┐   │
│  │  Auth      │  │  Monitor      │  │  History / Incident   │   │
│  │  Service   │  │  Service      │  │  Store               │   │
│  └─────┬──────┘  └──────┬────────┘  └──────────┬───────────┘   │
│        │                │                       │               │
│        └────────────────┴───────────────────────┘               │
│                         │                                       │
│                  ┌──────▼──────┐                                │
│                  │  pgxpool    │                                │
│                  │  (pgx/v5)   │                                │
│                  └──────┬──────┘                                │
└─────────────────────────┼───────────────────────────────────────┘
                          │  TCP
                  ┌───────▼────────┐
                  │  PostgreSQL    │
                  └────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  Scheduler  (separate goroutine, same process)                  │
│                                                                 │
│  polling loop (configurable interval, default 30s)              │
│       │                                                         │
│       ▼  fetch due monitors (next_check_at <= NOW())            │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Bounded worker pool  sem = make(chan struct{}, N)      │    │
│  │  for each monitor:                                      │    │
│  │    acquire sem → spawn goroutine → release on done      │    │
│  └───────────────────────┬────────────────────────────────┘    │
│                          │                                      │
│                  ┌───────▼────────┐                             │
│                  │  SSRF-safe     │                             │
│                  │  Checker       │                             │
│                  └───────┬────────┘                             │
│                          │  HTTP (resolved, validated IP)       │
│                  ┌───────▼────────┐                             │
│                  │  External      │                             │
│                  │  endpoint      │                             │
│                  └────────────────┘                             │
│                                                                 │
│  Checker result → history store, incident lifecycle,            │
│                   monitor.next_check_at update                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Package Layout

```
internal/
├── auth/
│   ├── model.go        User, RefreshToken domain types
│   ├── password.go     bcrypt hash/check, strength validation
│   ├── jwt.go          IssueToken / ParseToken (HS256)
│   ├── store.go        pgx-based user CRUD
│   └── service.go      Register, Login, UpdateProfile business logic
│
├── checker/
│   ├── transport.go    ssrfDialer — DNS resolve + IP validation
│   └── checker.go      Check() — performs the HTTP request
│
├── config/
│   └── config.go       env-var loading, validation
│
├── history/
│   ├── model.go        MonitorCheck domain type
│   └── store.go        RecordCheck, ListChecks, Availability, ResponseTimes
│
├── http/
│   ├── handlers/       One file per domain (auth, monitors, history, incidents, dashboard)
│   ├── middleware/     auth.go (JWT extraction), cors.go, logger.go
│   └── router.go       Route registration
│
├── incident/
│   ├── model.go        Incident domain type
│   └── store.go        OpenOrIncrement, ResolveIfOpen, ListIncidents
│
├── monitor/
│   ├── model.go        Monitor domain type, AllowedIntervals
│   ├── validate.go     ValidateCreate/Update, isObviouslyLocalhost
│   ├── store.go        DueMonitors, UpdateAfterCheck, full CRUD
│   └── service.go      Create, Update, PauseResume business logic
│
├── scheduler/
│   └── scheduler.go    Polling loop + bounded worker pool
│
└── storage/
    └── postgres.go     pgxpool factory, ping, Close
```

---

## Data Flow: Incoming API Request

```
1. Browser sends  POST /api/v1/monitors  + Bearer <jwt>
2. CORS middleware checks Origin header
3. Logger middleware records method/path
4. Auth middleware extracts and validates JWT → attaches userID to context
5. httprouter dispatches to handlers.CreateMonitor
6. Handler decodes JSON body, calls monitor.Service.Create
7. Service calls monitor.ValidateCreate (SSRF-safe URL check)
8. Service calls monitor.Store.Create → INSERT … RETURNING id
9. Handler writes 201 JSON response
```

## Data Flow: Scheduled Check

```
1. Scheduler polls:  SELECT … WHERE next_check_at <= NOW()
2. For each due monitor: acquire semaphore slot → launch goroutine
3. Goroutine calls Checker.Check(ctx, monitor)
4. ssrfDialer:
   a. net.DefaultResolver.LookupHost → []string IPs
   b. For each IP: isBlockedIP check (loopback, private, link-local, …)
   c. Dial to first validated IP only
5. HTTP GET with User-Agent: ServiceWatch/1.0, redirects disabled
6. Record result:
   a. history.Store.RecordCheck — INSERT into monitor_checks
   b. incident.Store.OpenOrIncrement / ResolveIfOpen
   c. monitor.Store.UpdateAfterCheck — updates status, next_check_at
7. Semaphore slot released
```

---

## Scheduler Design

The scheduler runs in a single goroutine started by `main()`. On each tick it:

1. Queries monitors due for checking (up to `workerCount * 2` rows).
2. Dispatches each into the bounded worker pool.
3. Waits for all goroutines to finish before sleeping until the next interval.

**Bounded worker pool** uses a semaphore channel:

```go
sem := make(chan struct{}, workerCount)
var wg sync.WaitGroup
for _, m := range dueMonitors {
    wg.Add(1)
    sem <- struct{}{}
    go func(m monitor.Monitor) {
        defer wg.Done()
        defer func() { <-sem }()
        s.runCheck(m)
    }(m)
}
wg.Wait()
```

**Graceful shutdown**: the scheduler's context is cancelled on SIGINT/SIGTERM. The polling loop exits on context cancellation, then `wg.Wait()` ensures all in-flight checks persist their results before `main()` returns.

**DB writes use `context.Background()`** (not the cancellable scheduler context) so results are always persisted even if shutdown was requested during a check.

---

## Authentication Flow

```
Register / Login
  → bcrypt hash/verify
  → IssueToken(user, secret, expiry) → signed JWT string
  → browser stores in localStorage as sw_token

Subsequent requests
  → Authorization: Bearer <jwt>
  → auth middleware: ParseToken → *Claims{UserID, Email}
  → claims attached to request context
  → handlers read userID from context for ownership checks
```

Tokens are stateless JWTs; there is no server-side session store. `refresh_tokens` table exists in the schema for future refresh-token support but is not used in the current implementation.

---

## SSRF Mitigation Layers

Two independent layers:

| Layer | Where | What it checks |
|---|---|---|
| 1. URL validation | monitor creation/update | Scheme (http/https only), credentials in URL, obvious localhost/::1, non-allowed intervals |
| 2. IP validation | dial time (every request) | Resolved IP is not loopback, link-local, private, multicast, or unspecified |

Layer 2 fires for every check, so it catches DNS changes between creation and check time. It does not fully mitigate DNS rebinding attacks (an attacker could return a public IP during validation, then a private IP during the actual connection). Network-level egress controls are recommended in production.

---

## Error Response Shape

All API errors follow:

```json
{
  "error": "human-readable message"
}
```

Successful responses follow:

```json
{
  "data": { … }
}
```

or for lists:

```json
{
  "data": [ … ],
  "pagination": { "page": 1, "page_size": 20, "total": 42, "total_pages": 3 }
}
```

---

## Frontend Architecture

The frontend is a single-page application with no server-side rendering.

```
src/
├── api/
│   ├── client.ts          fetch wrapper, ApiError, token injection
│   ├── auth.ts            register, login, getMe, updateProfile, changePassword
│   ├── monitors.ts        CRUD, pause, resume
│   ├── history.ts         checks, availability, responseTimes
│   └── dashboard.ts       getDashboard
│
├── components/
│   ├── ui/                StatusBadge, LoadingSpinner, Card, Button, Input, …
│   └── layout/            Layout (sidebar + header), Sidebar, Header
│
├── hooks/
│   ├── useAuth.ts         useCurrentUser, useLogin, useRegister, …
│   └── useMonitors.ts     useMonitors, useMonitor, useCreateMonitor, …
│
├── pages/                 LoginPage, RegisterPage, DashboardPage, …
├── types/                 Shared TS interfaces (Monitor, Check, Incident, …)
└── utils/                 formatDate, formatDuration, formatAvailability
```

TanStack Query manages all remote state. `staleTime: 60_000` means data is refetched at most once per minute per query key. The `retry: 1` setting retries failed queries once before surfacing the error.

Authentication state lives in `localStorage` (`sw_token`). On 401 responses the API client clears the token and redirects to `/login`.
