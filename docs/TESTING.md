# Testing

---

## Test Structure

```
internal/
├── auth/
│   ├── password_test.go   bcrypt hash/check, strength validation
│   └── jwt_test.go        IssueToken / ParseToken, expiry
├── checker/
│   └── checker_test.go    HTTP check behavior, SSRF blocking
└── monitor/
    └── validate_test.go   URL validation, interval checks
frontend/src/test/
├── setup.ts               jest-dom matchers
├── StatusBadge.test.tsx   UI component rendering
└── formatUtils.test.ts    Date/duration formatting utilities
```

---

## Running Tests

### Go

```bash
# All packages
go test ./... -count=1

# With race detector (recommended for CI)
go test ./... -count=1 -race -timeout 120s

# Single package
go test ./internal/checker/... -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Frontend

```bash
cd frontend

# Run all tests once
npm test
# or: npx vitest run

# Watch mode (interactive development)
npx vitest

# With coverage
npx vitest run --coverage
```

### Type checking and build

```bash
# Go
go vet ./...
gofmt -l ./cmd ./internal   # exits non-zero if any file needs formatting

# Frontend
npm run typecheck            # tsc --noEmit
npm run build                # production build via vite
```

---

## Test Results (2026-10-29)

```
go vet ./...                       PASS
gofmt ./cmd ./internal             PASS (all files formatted)

go test ./internal/auth/...        PASS  0.955s   13 tests
go test ./internal/checker/...     PASS  1.640s   18 tests
go test ./internal/monitor/...     PASS  0.549s   10 tests

vitest run                         PASS  15 tests in 2 suites
tsc --noEmit                       PASS
vite build                         PASS  815 KB bundle
```

---

## Go Test Coverage by Package

### `internal/auth`

| Test | What it covers |
|---|---|
| `TestHashPassword` | Valid hash and check round-trip |
| `TestHashPassword_Weak` | Rejects passwords below strength requirements |
| `TestCheckPassword_Wrong` | Returns `ErrInvalidCredentials` |
| `TestIssueToken` | Token string is non-empty, expiry is in the future |
| `TestParseToken` | Claims match the issued values |
| `TestParseToken_Expired` | Returns error for past-expiry token |
| `TestParseToken_WrongSecret` | Returns error for tampered signature |

### `internal/checker`

| Test | What it covers |
|---|---|
| `TestCheck_Success` | 200 response → `is_success=true`, `status_code=200` |
| `TestCheck_Timeout` | Context deadline exceeded → `is_success=false` |
| `TestCheck_Non200` | 500 response → `is_success=false` when expected_status=0 |
| `TestCheck_ExpectedStatus_Match` | expected_status=201 matches 201 |
| `TestCheck_ExpectedStatus_Mismatch` | expected_status=200, got 201 → failure |
| `TestCheck_Redirect` | 302 treated as failure (redirects disabled) |
| `TestSSRF_Loopback` | `127.0.0.1` blocked when `allowPrivateIPs=false` |
| `TestSSRF_PrivateIP` | `192.168.1.1` blocked |
| `TestSSRF_LinkLocal` | `169.254.169.254` (AWS metadata) blocked |
| `TestSSRF_IPv6Loopback` | `::1` blocked |

### `internal/monitor`

| Test | What it covers |
|---|---|
| `TestValidate_ValidHTTPS` | Accepts a well-formed https URL |
| `TestValidate_ValidHTTP` | Accepts a well-formed http URL |
| `TestValidate_RejectsLocalhost` | `http://localhost/` rejected |
| `TestValidate_RejectsLoopback` | `http://127.0.0.1/` rejected |
| `TestValidate_RejectsIPv6Loopback` | `http://[::1]/path` rejected |
| `TestValidate_RejectsCredentials` | `http://user:pass@host/` rejected |
| `TestValidate_RejectsNonHTTP` | `ftp://example.com` rejected |
| `TestValidate_RejectsInvalidInterval` | `interval_seconds=45` rejected |
| `TestValidate_RejectsInvalidTimeout` | `timeout_seconds=3` rejected |
| `TestValidate_RejectsInvalidStatus` | `expected_status=99` rejected |

---

## Frontend Test Coverage

### `StatusBadge.test.tsx`

| Test | What it covers |
|---|---|
| Renders "Up" for up status | Green badge text and CSS class |
| Renders "Down" for down status | Red badge |
| Renders "Pending" for pending status | Yellow badge |
| Renders "Paused" for paused status | Gray badge |
| Has colored dot indicator | Dot element present in DOM |

### `formatUtils.test.ts`

| Test | What it covers |
|---|---|
| Formats seconds < 60 | "45s" |
| Formats seconds = 60 | "1m 0s" |
| Formats minutes | "2m 30s" |
| Formats zero | "0s" |
| Formats availability to 2 decimal places | "99.86%" |
| Formats perfect availability | "100.00%" |
| Formats zero availability | "0.00%" |
| Formats milliseconds short | "142ms" |
| Formats milliseconds > 1000 | "1.2s" |
| Formats null/undefined gracefully | "—" |

---

## Testing Philosophy

**Unit tests only.** The test suite uses `httptest.NewServer` to stand up real HTTP test servers instead of mocking. This gives more confidence that the SSRF dialer and checker interact correctly.

**No database tests.** Store-layer tests would require a test PostgreSQL instance. The trade-off was accepted: the stores are thin wrappers around parameterized queries, and the schema is verified by running migrations manually.

**Race detector in CI.** `go test -race` catches data races in the scheduler and checker. All three packages pass with the race detector.

**Test isolation.** Each checker test creates an independent `httptest.Server` and tears it down with `t.Cleanup`. Tests do not share state.

---

## Adding Tests

### Go

Place test files adjacent to the package they test (`foo_test.go` in the same directory as `foo.go`). Use the `_test` package suffix for black-box tests:

```go
package checker_test

import (
    "testing"
    "github.com/amsoft/servicewatch/internal/checker"
)
```

### Frontend

Place test files in `src/test/`. Import testing utilities from `@testing-library/react` and `vitest`. The `setup.ts` file extends `expect` with `jest-dom` matchers.

```typescript
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
```
