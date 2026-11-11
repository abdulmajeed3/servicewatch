package monitor

import (
	"errors"
	"time"
)

// Status represents the current health state of a monitor.
type Status string

const (
	StatusUp      Status = "up"
	StatusDown    Status = "down"
	StatusPending Status = "pending"
	StatusPaused  Status = "paused"
)

// AllowedIntervals lists the supported check intervals in seconds.
var AllowedIntervals = []int{60, 300, 600, 1800, 3600}

// Monitor is the core entity representing a monitored endpoint.
type Monitor struct {
	ID                  int64      `json:"id"`
	UserID              int64      `json:"user_id"`
	Name                string     `json:"name"`
	URL                 string     `json:"url"`
	CheckInterval       int        `json:"check_interval"`
	RequestTimeout      int        `json:"request_timeout"`
	ExpectedStatus      int        `json:"expected_status"`
	Status              Status     `json:"status"`
	IsActive            bool       `json:"is_active"`
	NextCheckAt         time.Time  `json:"next_check_at"`
	LastCheckedAt       *time.Time `json:"last_checked_at"`
	LastStatusCode      *int       `json:"last_status_code"`
	LastResponseMs      *int       `json:"last_response_ms"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	Notes               string     `json:"notes"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// CreateRequest is the payload for creating a new monitor.
type CreateRequest struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	CheckInterval  int    `json:"check_interval"`
	RequestTimeout int    `json:"request_timeout"`
	ExpectedStatus int    `json:"expected_status"`
	Notes          string `json:"notes"`
}

// UpdateRequest is the payload for editing an existing monitor.
type UpdateRequest struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	CheckInterval  int    `json:"check_interval"`
	RequestTimeout int    `json:"request_timeout"`
	ExpectedStatus int    `json:"expected_status"`
	Notes          string `json:"notes"`
}

// ListFilter carries search/filter parameters for the monitor list.
type ListFilter struct {
	Search string
	Status Status
	Page   int
	Limit  int
}

// ListResult wraps a paginated list of monitors.
type ListResult struct {
	Monitors   []*Monitor `json:"monitors"`
	TotalCount int        `json:"total_count"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
}

// DashboardStats contains aggregated counts for the dashboard.
type DashboardStats struct {
	Total   int `json:"total"`
	Up      int `json:"up"`
	Down    int `json:"down"`
	Paused  int `json:"paused"`
	Pending int `json:"pending"`
}

var (
	ErrNotFound     = errors.New("monitor not found")
	ErrForbidden    = errors.New("access denied")
	ErrInvalidInput = errors.New("invalid input")
)
