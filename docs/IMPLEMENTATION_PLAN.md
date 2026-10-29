# Implementation Plan

This document records the implementation phases used to build ServiceWatch from scratch.

---

## Status Summary

| Phase | Description | Status |
|---|---|---|
| 0 | Inspect environment | ✅ Complete |
| 1 | Foundation | ✅ Complete |
| 2 | Auth + Monitor core | ✅ Complete |
| 3 | Monitoring engine | ✅ Complete |
| 4 | History + Incidents | ✅ Complete |
| 5 | Frontend | ✅ Complete |
| 6 | Testing | ✅ Complete |
| 7 | Quality checks | ✅ Complete |
| 8 | Documentation | ✅ Complete |
| 9 | Repository + CI | ✅ Complete |

---

## Phase 0 — Inspect Environment

- Confirmed Go 1.23+ installed (`go version`)
- Confirmed Node.js 22+ and npm installed
- Confirmed PostgreSQL 18 installed and running as a Windows service
- Confirmed working directory: `C:\Users\AMSoft\source\repos\Github-Projects\ServiceWatch`
- Selected module path: `github.com/amsoft/servicewatch`

---

## Phase 1 — Foundation

- Initialized Go module (`go.mod`)
- Added dependencies: `golang-jwt/jwt/v5`, `pgx/v5`, `httprouter`, `golang.org/x/crypto`
- Created `internal/config/config.go` with environment-variable loading
- Created `internal/storage/postgres.go` with `pgxpool` factory
- Created `migrations/001_initial_schema.sql` (all tables + indexes)
- Created `migrations/002_seed_data.sql` (demo user + seed monitors)
- Created `cmd/server/main.go` entry point

---

## Phase 2 — Auth + Monitor Core

- `internal/auth/password.go` — bcrypt hash/check, strength validation
- `internal/auth/jwt.go` — HS256 token issuance and parsing
- `internal/auth/model.go` — User domain type
- `internal/auth/store.go` — pgx user CRUD
- `internal/auth/service.go` — Register, Login, UpdateProfile, ChangePassword
- `internal/monitor/model.go` — Monitor domain type, AllowedIntervals
- `internal/monitor/validate.go` — SSRF-aware URL validation
- `internal/monitor/store.go` — Monitor CRUD, DueMonitors, UpdateAfterCheck
- `internal/monitor/service.go` — Create, Update, PauseResume
- `internal/http/middleware/` — auth.go (JWT), cors.go, logger.go
- `internal/http/handlers/auth.go` — Register, Login, Me, UpdateProfile, ChangePassword
- `internal/http/handlers/monitors.go` — List, Create, Get, Update, Delete, Pause, Resume
- `internal/http/router.go` — all routes wired up

---

## Phase 3 — Monitoring Engine

- `internal/checker/transport.go` — `ssrfDialer`, `isBlockedIP`, `NewTransport`
- `internal/checker/checker.go` — `Check()` with timeout, User-Agent, redirect disabled, body limit
- `internal/scheduler/scheduler.go` — bounded worker pool, polling loop, graceful shutdown

---

## Phase 4 — History + Incidents

- `internal/history/model.go` — MonitorCheck domain type
- `internal/history/store.go` — RecordCheck, ListChecks, Availability, ResponseTimes
- `internal/incident/model.go` — Incident domain type
- `internal/incident/store.go` — OpenOrIncrement, ResolveIfOpen, ListIncidents
- `internal/http/handlers/history.go` — checks, availability, response-times, incidents
- `internal/http/handlers/dashboard.go` — aggregate stats

---

## Phase 5 — Frontend

Scaffolded with `npm create vite@latest frontend -- --template react-ts`.

- `frontend/vite.config.ts` — Tailwind plugin, `/api` proxy, vitest config
- `frontend/src/api/` — typed API client (auth, monitors, history, dashboard)
- `frontend/src/types/` — shared TypeScript interfaces
- `frontend/src/utils/` — formatDate, formatDuration, formatAvailability
- `frontend/src/components/ui/` — StatusBadge, LoadingSpinner, Card, Button, Input, etc.
- `frontend/src/components/layout/` — Layout, Sidebar, Header
- `frontend/src/hooks/` — useAuth, useMonitors (TanStack Query)
- Pages: Login, Register, Dashboard, MonitorList, MonitorDetail, MonitorCreate, MonitorEdit, History, Profile, NotFound
- Routing: React Router v7 with protected routes

---

## Phase 6 — Testing

- `internal/auth/password_test.go` — 7 tests
- `internal/auth/jwt_test.go` — 6 tests
- `internal/checker/checker_test.go` — 18 tests (includes SSRF blocking)
- `internal/monitor/validate_test.go` — 10 tests
- `frontend/src/test/StatusBadge.test.tsx` — 5 tests
- `frontend/src/test/formatUtils.test.ts` — 10 tests
- Created `frontend/tsconfig.test.json` (separate tsconfig for test files)

---

## Phase 7 — Quality Checks

All checks passed on 2026-10-29:

```
go vet ./...                  PASS
gofmt ./cmd ./internal        PASS  (13 files reformatted, then clean)
go test ./... -race           PASS  (41 tests)
Frontend: vitest run          PASS  (15 tests)
Frontend: tsc --noEmit        PASS
Frontend: vite build          PASS
```

Bugs found and fixed during this phase:

1. `TestValidate_RejectsLocalhost` for `http://::1/path` — fixed by changing to `http://[::1]/path` (correct IPv6 URL format)
2. `checker.go` naming conflict — `type net struct{}` shadowed the `net` package import; renamed to avoid conflict
3. Unused imports in `HistoryPage.tsx` and `MonitorListPage.tsx` — removed
4. `vite.config.ts` imported `defineConfig` from `vite` instead of `vitest/config` — fixed import
5. Test files had jest-dom types conflicting with production tsconfig — excluded test files from `tsconfig.app.json`, created `tsconfig.test.json`
6. Missing `@testing-library/dom` — installed with `--legacy-peer-deps`

---

## Phase 8 — Documentation

- `README.md` — comprehensive root README (features, architecture, quick start, API summary, configuration, testing, known limitations)
- `docs/ARCHITECTURE.md` — component overview, package layout, data flows, scheduler design, auth flow, SSRF layers, error shapes, frontend architecture
- `docs/DATABASE.md` — full schema, relationships, availability/response-time queries, retention guidance, connection pool
- `docs/API.md` — every endpoint with request/response examples
- `docs/SECURITY.md` — SSRF threat model, defense-in-depth layers, known limitations (DNS rebinding), auth design, input validation
- `docs/TESTING.md` — how to run tests, test coverage table, testing philosophy
- `docs/DEPLOYMENT.md` — build, systemd service, nginx/Caddy config, TLS, security checklist
- `docs/TROUBLESHOOTING.md` — common errors and fixes
- `docs/IMPLEMENTATION_PLAN.md` — this file

---

## Phase 9 — Repository + CI

- `.env.example` — all environment variables documented
- `.gitignore` — Go, Node, editor, Docker, database exclusions
- `.editorconfig` — tabs for Go, 2-space indent for TS/JSON/YAML
- `docker-compose.yml` — PostgreSQL 17-alpine container
- `.github/workflows/ci.yml` — backend (gofmt, go vet, go test -race) + frontend (typecheck, vitest, vite build) jobs
- `CONTRIBUTING.md`, `CHANGELOG.md`, `CODE_OF_CONDUCT.md`, `LICENSE`
- `.github/ISSUE_TEMPLATE/` — bug report and feature request templates
- `.prettierrc` — frontend formatting configuration

---

## Definition of Done

From the specification, all criteria met:

- [x] `go vet ./...` passes with no output
- [x] `gofmt -l ./cmd ./internal` returns no files
- [x] `go test ./...` all pass including race detector
- [x] Frontend `tsc --noEmit` passes
- [x] Frontend `vitest run` all pass
- [x] Frontend `vite build` succeeds
- [x] README.md covers all required sections
- [x] SSRF protection documented and tested
- [x] No secrets in repository
- [x] All migrations are idempotent
- [x] Graceful shutdown implemented
- [x] Ownership isolation enforced in all handlers
