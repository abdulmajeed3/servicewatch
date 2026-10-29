package history

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store handles monitoring check persistence and queries.
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// Record saves a new monitoring check result.
func (s *Store) Record(ctx context.Context, monitorID, userID int64, isSuccess bool, statusCode, responseMs int, failureReason string) error {
	var scPtr *int
	if statusCode != 0 {
		scPtr = &statusCode
	}
	var rmPtr *int
	if responseMs >= 0 {
		rmPtr = &responseMs
	}
	var frPtr *string
	if failureReason != "" {
		frPtr = &failureReason
	}

	_, err := s.db.Exec(ctx,
		`INSERT INTO monitor_checks (monitor_id, user_id, is_success, status_code, response_ms, failure_reason)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		monitorID, userID, isSuccess, scPtr, rmPtr, frPtr,
	)
	if err != nil {
		return fmt.Errorf("record check: %w", err)
	}
	return nil
}

// List returns paginated check history for a monitor owned by the given user.
func (s *Store) List(ctx context.Context, monitorID, userID int64, f ListFilter) (*ListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 200 {
		f.Limit = 50
	}
	offset := (f.Page - 1) * f.Limit

	args := []any{monitorID, userID}
	argIdx := 3
	conds := "monitor_id = $1 AND user_id = $2"

	if f.StartTime != nil {
		conds += fmt.Sprintf(" AND checked_at >= $%d", argIdx)
		args = append(args, *f.StartTime)
		argIdx++
	}
	if f.EndTime != nil {
		conds += fmt.Sprintf(" AND checked_at <= $%d", argIdx)
		args = append(args, *f.EndTime)
		argIdx++
	}
	if f.OnlyFails {
		conds += " AND is_success = FALSE"
	}

	var total int
	if err := s.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM monitor_checks WHERE %s", conds),
		args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count checks: %w", err)
	}

	listArgs := append(args, f.Limit, offset)
	rows, err := s.db.Query(ctx,
		fmt.Sprintf(`SELECT id, monitor_id, user_id, checked_at, is_success, status_code, response_ms, failure_reason, created_at
		 FROM monitor_checks
		 WHERE %s
		 ORDER BY checked_at DESC
		 LIMIT $%d OFFSET $%d`, conds, argIdx, argIdx+1),
		listArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("list checks: %w", err)
	}
	defer rows.Close()

	checks := make([]*Check, 0)
	for rows.Next() {
		var c Check
		if err := rows.Scan(&c.ID, &c.MonitorID, &c.UserID, &c.CheckedAt, &c.IsSuccess,
			&c.StatusCode, &c.ResponseMs, &c.FailureReason, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan check: %w", err)
		}
		checks = append(checks, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checks: %w", err)
	}

	return &ListResult{Checks: checks, TotalCount: total, Page: f.Page, Limit: f.Limit}, nil
}

// Availability calculates the uptime percentage over a time window for a monitor.
func (s *Store) Availability(ctx context.Context, monitorID, userID int64, windowHours int) (*AvailabilityStat, error) {
	since := time.Now().UTC().Add(-time.Duration(windowHours) * time.Hour)
	var total, successful int
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE is_success = TRUE)
		 FROM monitor_checks
		 WHERE monitor_id = $1 AND user_id = $2 AND checked_at >= $3`,
		monitorID, userID, since,
	).Scan(&total, &successful)
	if err != nil {
		return nil, fmt.Errorf("availability: %w", err)
	}

	var pct float64
	if total > 0 {
		pct = float64(successful) / float64(total) * 100.0
	}

	return &AvailabilityStat{
		WindowHours:  windowHours,
		Total:        total,
		Successful:   successful,
		Availability: pct,
	}, nil
}

// ResponseTimeSeries returns recent response time data points for charting.
// At most 200 points; caller should pass a sensible window.
func (s *Store) ResponseTimeSeries(ctx context.Context, monitorID, userID int64, since time.Time) ([]*ResponseTimeSeries, error) {
	rows, err := s.db.Query(ctx,
		`SELECT checked_at, COALESCE(response_ms, 0), is_success
		 FROM monitor_checks
		 WHERE monitor_id = $1 AND user_id = $2 AND checked_at >= $3
		 ORDER BY checked_at ASC
		 LIMIT 200`,
		monitorID, userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("response time series: %w", err)
	}
	defer rows.Close()

	var points []*ResponseTimeSeries
	for rows.Next() {
		var p ResponseTimeSeries
		if err := rows.Scan(&p.CheckedAt, &p.ResponseMs, &p.IsSuccess); err != nil {
			return nil, fmt.Errorf("scan series: %w", err)
		}
		points = append(points, &p)
	}
	return points, rows.Err()
}

// RecentFailures returns the last N failed checks across all monitors for a user.
func (s *Store) RecentFailures(ctx context.Context, userID int64, limit int) ([]*Check, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, monitor_id, user_id, checked_at, is_success, status_code, response_ms, failure_reason, created_at
		 FROM monitor_checks
		 WHERE user_id = $1 AND is_success = FALSE
		 ORDER BY checked_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent failures: %w", err)
	}
	defer rows.Close()

	var checks []*Check
	for rows.Next() {
		var c Check
		if err := rows.Scan(&c.ID, &c.MonitorID, &c.UserID, &c.CheckedAt, &c.IsSuccess,
			&c.StatusCode, &c.ResponseMs, &c.FailureReason, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan failure: %w", err)
		}
		checks = append(checks, &c)
	}
	return checks, rows.Err()
}
