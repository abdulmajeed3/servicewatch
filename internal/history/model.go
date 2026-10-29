package history

import "time"

// Check represents a single recorded monitoring result.
type Check struct {
	ID            int64     `json:"id"`
	MonitorID     int64     `json:"monitor_id"`
	UserID        int64     `json:"user_id"`
	CheckedAt     time.Time `json:"checked_at"`
	IsSuccess     bool      `json:"is_success"`
	StatusCode    *int      `json:"status_code"`
	ResponseMs    *int      `json:"response_ms"`
	FailureReason *string   `json:"failure_reason"`
	CreatedAt     time.Time `json:"created_at"`
}

// ListFilter carries pagination and time-range parameters.
type ListFilter struct {
	Page      int
	Limit     int
	StartTime *time.Time
	EndTime   *time.Time
	OnlyFails bool
}

// ListResult wraps a paginated list of checks.
type ListResult struct {
	Checks     []*Check `json:"checks"`
	TotalCount int      `json:"total_count"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
}

// AvailabilityStat holds the availability percentage for a time window.
type AvailabilityStat struct {
	WindowHours  int     `json:"window_hours"`
	Total        int     `json:"total"`
	Successful   int     `json:"successful"`
	Availability float64 `json:"availability"`
}

// ResponseTimeSeries is a single data point for charting.
type ResponseTimeSeries struct {
	CheckedAt  time.Time `json:"checked_at"`
	ResponseMs int       `json:"response_ms"`
	IsSuccess  bool      `json:"is_success"`
}
