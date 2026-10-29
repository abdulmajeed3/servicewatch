package incident

import "time"

// Incident tracks an outage event for a monitor.
type Incident struct {
	ID            int64      `json:"id"`
	MonitorID     int64      `json:"monitor_id"`
	UserID        int64      `json:"user_id"`
	StartedAt     time.Time  `json:"started_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	FailureCount  int        `json:"failure_count"`
	LastFailureAt time.Time  `json:"last_failure_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ListFilter carries pagination for incident lists.
type ListFilter struct {
	Page  int
	Limit int
	Open  *bool // nil = all, true = only open, false = only resolved
}

// ListResult wraps a paginated incident list.
type ListResult struct {
	Incidents  []*Incident `json:"incidents"`
	TotalCount int         `json:"total_count"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
}
