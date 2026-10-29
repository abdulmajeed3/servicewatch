# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Added
- Initial release: full-stack service monitoring platform

### Backend
- Go HTTP API under `/api/v1` using `httprouter`
- JWT authentication (HS256) with bcrypt password hashing
- Monitor CRUD with SSRF-safe URL validation
- Custom `ssrfDialer` blocking loopback, private, link-local, multicast, and unspecified IP addresses
- Bounded worker pool scheduler with configurable concurrency
- Check history persistence with availability percentage calculation
- Incident open/resolve lifecycle (one open incident per monitor)
- Dashboard aggregate endpoint
- Graceful shutdown with in-flight check completion
- Health endpoints: `/health/live` and `/health/ready`
- Structured JSON logging

### Frontend
- React 19 + TypeScript SPA built with Vite 8
- TanStack Query for data fetching with 60-second stale time
- React Router v7 protected routes
- React Hook Form + Zod for form validation
- Tailwind CSS v4 styling
- Recharts response-time chart on monitor detail page
- Pages: Login, Register, Dashboard, Monitor List, Monitor Detail, Create/Edit Monitor, History, Profile, 404
- Status badges (up/down/pending/paused) with colored indicators

### Database
- PostgreSQL schema with explicit SQL migrations
- Tables: `users`, `monitors`, `monitor_checks`, `incidents`, `refresh_tokens`, `schema_migrations`
- Partial index on `monitors.next_check_at WHERE is_active = TRUE`
- Development seed data (demo account + 3 seed monitors)

### Testing
- 41 Go unit tests across auth, checker, and monitor packages
- Race detector passing on all packages
- 15 frontend tests (component rendering + utility functions)
- Frontend type check and production build passing

### CI
- GitHub Actions CI: Go (gofmt, go vet, go test -race) + Frontend (typecheck, vitest, vite build)

### Documentation
- Comprehensive README with quick start, API reference, configuration
- docs/ARCHITECTURE.md, docs/DATABASE.md, docs/API.md
- docs/SECURITY.md, docs/TESTING.md, docs/DEPLOYMENT.md
- docs/TROUBLESHOOTING.md, docs/IMPLEMENTATION_PLAN.md
