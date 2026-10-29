# API Reference

All endpoints are under the prefix `/api/v1`. The server runs on port 8080 by default.

---

## Response Envelope

**Success (single object):**
```json
{ "data": { … } }
```

**Success (list):**
```json
{
  "data": [ … ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

**Error:**
```json
{ "error": "human-readable message" }
```

---

## Authentication

Authenticated endpoints require:

```
Authorization: Bearer <jwt_token>
```

Tokens are obtained from `POST /api/v1/auth/login` or `POST /api/v1/auth/register`.

---

## Health

### `GET /health/live`

Liveness check. Returns 200 while the process is running.

**Response:** `200 OK` — body: `ok`

### `GET /health/ready`

Readiness check. Returns 200 if the database is reachable.

**Response:** `200 OK` — body: `ok`  
**Error:** `503 Service Unavailable`

---

## Auth Endpoints

### `POST /api/v1/auth/register`

Create a new user account.

**Request body:**
```json
{
  "email": "user@example.com",
  "password": "Secure1234!",
  "display_name": "Jane Smith"
}
```

**Password rules:** 8–128 characters, at least one uppercase letter, one lowercase letter, one digit.

**Response:** `201 Created`
```json
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9…",
    "expires_at": "2026-10-30T12:00:00Z",
    "user": {
      "id": "a1b2c3d4-…",
      "email": "user@example.com",
      "display_name": "Jane Smith",
      "created_at": "2026-10-29T12:00:00Z"
    }
  }
}
```

**Errors:** `400` (validation), `409` (email already registered)

---

### `POST /api/v1/auth/login`

Authenticate with email and password.

**Request body:**
```json
{
  "email": "user@example.com",
  "password": "Secure1234!"
}
```

**Response:** `200 OK` — same shape as register.

**Errors:** `401` (invalid credentials — same message for wrong email or wrong password to prevent enumeration)

---

### `GET /api/v1/auth/me` 🔒

Return the authenticated user's profile.

**Response:** `200 OK`
```json
{
  "data": {
    "id": "a1b2c3d4-…",
    "email": "user@example.com",
    "display_name": "Jane Smith",
    "created_at": "2026-10-29T12:00:00Z",
    "updated_at": "2026-10-29T12:00:00Z"
  }
}
```

---

### `PATCH /api/v1/auth/me` 🔒

Update display name or email.

**Request body** (all fields optional):
```json
{
  "display_name": "Jane Doe",
  "email": "new@example.com"
}
```

**Response:** `200 OK` — updated user object wrapped in `{ "data": … }`.

**Errors:** `400` (validation), `409` (email taken)

---

### `POST /api/v1/auth/change-password` 🔒

Change the authenticated user's password.

**Request body:**
```json
{
  "current_password": "OldPass1!",
  "new_password": "NewPass2@"
}
```

**Response:** `200 OK`

**Errors:** `400` (new password fails strength check), `401` (current password wrong)

---

## Monitor Endpoints

### `GET /api/v1/monitors` 🔒

List all monitors owned by the authenticated user.

**Query parameters:**

| Parameter | Default | Description |
|---|---|---|
| `page` | `1` | Page number |
| `page_size` | `20` | Results per page (max 100) |
| `status` | — | Filter by status: `up`, `down`, `pending`, `paused` |
| `search` | — | Case-insensitive substring match on name/URL |

**Response:** `200 OK`
```json
{
  "data": [
    {
      "id": "…",
      "name": "API Gateway",
      "url": "https://api.example.com/health",
      "interval_seconds": 300,
      "timeout_seconds": 30,
      "expected_status": 200,
      "is_active": true,
      "status": "up",
      "consecutive_failures": 0,
      "last_checked_at": "2026-10-29T11:55:00Z",
      "next_check_at": "2026-10-29T12:00:00Z",
      "last_response_ms": 142,
      "created_at": "2026-10-01T10:00:00Z",
      "updated_at": "2026-10-29T11:55:00Z"
    }
  ],
  "pagination": { "page": 1, "page_size": 20, "total": 5, "total_pages": 1 }
}
```

---

### `POST /api/v1/monitors` 🔒

Create a new monitor.

**Request body:**
```json
{
  "name": "API Gateway",
  "url": "https://api.example.com/health",
  "interval_seconds": 300,
  "timeout_seconds": 30,
  "expected_status": 200
}
```

| Field | Required | Constraints |
|---|---|---|
| `name` | Yes | 1–200 chars |
| `url` | Yes | https/http, no credentials, not localhost/private |
| `interval_seconds` | No | One of: 60, 300, 600, 1800, 3600 (default: 300) |
| `timeout_seconds` | No | 5–120 (default: 30) |
| `expected_status` | No | 0 (any 2xx) or 100–599 (default: 0) |

**Response:** `201 Created` — the created monitor wrapped in `{ "data": … }`.

**Errors:** `400` (validation), `422` (SSRF-unsafe URL)

---

### `GET /api/v1/monitors/:id` 🔒

Get a single monitor by ID.

**Response:** `200 OK` — monitor object wrapped in `{ "data": … }`.

**Errors:** `404` (not found or not owned by caller)

---

### `PUT /api/v1/monitors/:id` 🔒

Replace a monitor's configuration. All fields are required (full replacement).

**Request body:** same fields as `POST /api/v1/monitors`.

**Response:** `200 OK` — updated monitor wrapped in `{ "data": … }`.

**Errors:** `400`, `404`, `422`

---

### `DELETE /api/v1/monitors/:id` 🔒

Delete a monitor and all its history.

**Response:** `204 No Content`

**Errors:** `404`

---

### `POST /api/v1/monitors/:id/pause` 🔒

Pause a monitor. Paused monitors are skipped by the scheduler.

**Response:** `200 OK` — updated monitor wrapped in `{ "data": … }`.

---

### `POST /api/v1/monitors/:id/resume` 🔒

Resume a paused monitor. Sets `next_check_at = NOW()` so it is checked on the next scheduler tick.

**Response:** `200 OK` — updated monitor wrapped in `{ "data": … }`.

---

## History Endpoints

### `GET /api/v1/monitors/:id/checks` 🔒

Paginated check history.

**Query parameters:**

| Parameter | Default | Description |
|---|---|---|
| `page` | `1` | Page number |
| `page_size` | `50` | Results per page (max 200) |

**Response:** `200 OK`
```json
{
  "data": [
    {
      "id": "…",
      "monitor_id": "…",
      "checked_at": "2026-10-29T12:00:00Z",
      "is_success": true,
      "status_code": 200,
      "response_ms": 142,
      "error_message": null
    }
  ],
  "pagination": { "page": 1, "page_size": 50, "total": 288, "total_pages": 6 }
}
```

---

### `GET /api/v1/monitors/:id/availability` 🔒

Availability percentage over a time window.

**Query parameters:**

| Parameter | Default | Description |
|---|---|---|
| `window` | `24h` | Time window: `1h`, `6h`, `24h`, `7d`, `30d` |

**Response:** `200 OK`
```json
{
  "data": {
    "availability_pct": 99.86,
    "total_checks": 288,
    "successful_checks": 287,
    "window": "24h"
  }
}
```

---

### `GET /api/v1/monitors/:id/response-times` 🔒

Response time series for charting.

**Query parameters:**

| Parameter | Default | Description |
|---|---|---|
| `window` | `24h` | Same options as availability |

**Response:** `200 OK`
```json
{
  "data": [
    { "checked_at": "2026-10-29T11:00:00Z", "response_ms": 138 },
    { "checked_at": "2026-10-29T11:05:00Z", "response_ms": 145 }
  ]
}
```

Failed checks (where `response_ms` is null) are excluded from this series.

---

### `GET /api/v1/monitors/:id/incidents` 🔒

Incident history for a monitor.

**Query parameters:** `page`, `page_size` (same as checks).

**Response:** `200 OK`
```json
{
  "data": [
    {
      "id": "…",
      "monitor_id": "…",
      "started_at": "2026-10-28T03:15:00Z",
      "resolved_at": "2026-10-28T03:45:00Z",
      "failure_count": 6,
      "duration_seconds": 1800
    }
  ],
  "pagination": { … }
}
```

`resolved_at` is `null` for open (ongoing) incidents. `duration_seconds` is omitted for open incidents.

---

## Dashboard

### `GET /api/v1/dashboard` 🔒

Aggregate statistics for the authenticated user.

**Response:** `200 OK`
```json
{
  "data": {
    "total_monitors": 5,
    "monitors_up": 4,
    "monitors_down": 1,
    "monitors_paused": 0,
    "monitors_pending": 0,
    "overall_availability_24h": 98.6,
    "recent_failures": [
      {
        "monitor_id": "…",
        "monitor_name": "Payments API",
        "checked_at": "2026-10-29T11:55:00Z",
        "error_message": "connection refused"
      }
    ],
    "open_incidents": [
      {
        "id": "…",
        "monitor_id": "…",
        "monitor_name": "Payments API",
        "started_at": "2026-10-29T11:50:00Z",
        "failure_count": 2
      }
    ]
  }
}
```

`recent_failures` returns the last 5 failed checks across all monitors. `open_incidents` returns all currently open incidents.

---

## Error Codes

| HTTP Status | When |
|---|---|
| 400 Bad Request | Malformed JSON, validation failure |
| 401 Unauthorized | Missing or invalid JWT |
| 403 Forbidden | Attempting to access another user's resource |
| 404 Not Found | Resource does not exist or belongs to another user |
| 409 Conflict | Unique constraint (e.g., email already registered) |
| 422 Unprocessable Entity | Semantically invalid input (e.g., SSRF-unsafe URL) |
| 500 Internal Server Error | Unexpected server error |
| 503 Service Unavailable | Database not reachable (readiness endpoint only) |
