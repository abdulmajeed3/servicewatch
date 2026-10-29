package monitor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store handles monitor persistence.
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// Create inserts a new monitor and returns it.
func (s *Store) Create(ctx context.Context, userID int64, req *CreateRequest) (*Monitor, error) {
	var m Monitor
	err := s.db.QueryRow(ctx,
		`INSERT INTO monitors
		   (user_id, name, url, check_interval, request_timeout, expected_status)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id, user_id, name, url, check_interval, request_timeout, expected_status,
		           status, is_active, next_check_at, last_checked_at, last_status_code,
		           last_response_ms, consecutive_failures, created_at, updated_at`,
		userID, req.Name, req.URL, req.CheckInterval, req.RequestTimeout, req.ExpectedStatus,
	).Scan(
		&m.ID, &m.UserID, &m.Name, &m.URL, &m.CheckInterval, &m.RequestTimeout, &m.ExpectedStatus,
		&m.Status, &m.IsActive, &m.NextCheckAt, &m.LastCheckedAt, &m.LastStatusCode,
		&m.LastResponseMs, &m.ConsecutiveFailures, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create monitor: %w", err)
	}
	return &m, nil
}

// GetByID returns a monitor by ID without ownership check.
func (s *Store) GetByID(ctx context.Context, id int64) (*Monitor, error) {
	var m Monitor
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, name, url, check_interval, request_timeout, expected_status,
		        status, is_active, next_check_at, last_checked_at, last_status_code,
		        last_response_ms, consecutive_failures, created_at, updated_at
		 FROM monitors WHERE id = $1`,
		id,
	).Scan(
		&m.ID, &m.UserID, &m.Name, &m.URL, &m.CheckInterval, &m.RequestTimeout, &m.ExpectedStatus,
		&m.Status, &m.IsActive, &m.NextCheckAt, &m.LastCheckedAt, &m.LastStatusCode,
		&m.LastResponseMs, &m.ConsecutiveFailures, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get monitor: %w", err)
	}
	return &m, nil
}

// GetByIDForUser returns a monitor owned by the given user, or ErrNotFound / ErrForbidden.
func (s *Store) GetByIDForUser(ctx context.Context, id, userID int64) (*Monitor, error) {
	m, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m.UserID != userID {
		return nil, ErrNotFound // hide existence from other users
	}
	return m, nil
}

// List returns a paginated list of monitors for a user with optional filtering.
func (s *Store) List(ctx context.Context, userID int64, f ListFilter) (*ListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}
	offset := (f.Page - 1) * f.Limit

	conditions := []string{"user_id = $1"}
	args := []any{userID}
	argIdx := 2

	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(f.Status))
		argIdx++
	}
	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR url ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	err := s.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM monitors %s", where),
		args...,
	).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count monitors: %w", err)
	}

	countArgs := append(args, f.Limit, offset)
	rows, err := s.db.Query(ctx,
		fmt.Sprintf(`SELECT id, user_id, name, url, check_interval, request_timeout, expected_status,
		        status, is_active, next_check_at, last_checked_at, last_status_code,
		        last_response_ms, consecutive_failures, created_at, updated_at
		 FROM monitors %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1),
		countArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]*Monitor, 0)
	for rows.Next() {
		var m Monitor
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.Name, &m.URL, &m.CheckInterval, &m.RequestTimeout, &m.ExpectedStatus,
			&m.Status, &m.IsActive, &m.NextCheckAt, &m.LastCheckedAt, &m.LastStatusCode,
			&m.LastResponseMs, &m.ConsecutiveFailures, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan monitor: %w", err)
		}
		monitors = append(monitors, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monitors: %w", err)
	}

	return &ListResult{
		Monitors:   monitors,
		TotalCount: total,
		Page:       f.Page,
		Limit:      f.Limit,
	}, nil
}

// Update modifies monitor configuration fields.
func (s *Store) Update(ctx context.Context, id int64, req *UpdateRequest) (*Monitor, error) {
	var m Monitor
	err := s.db.QueryRow(ctx,
		`UPDATE monitors
		 SET name = $1, url = $2, check_interval = $3, request_timeout = $4,
		     expected_status = $5, updated_at = NOW()
		 WHERE id = $6
		 RETURNING id, user_id, name, url, check_interval, request_timeout, expected_status,
		           status, is_active, next_check_at, last_checked_at, last_status_code,
		           last_response_ms, consecutive_failures, created_at, updated_at`,
		req.Name, req.URL, req.CheckInterval, req.RequestTimeout, req.ExpectedStatus, id,
	).Scan(
		&m.ID, &m.UserID, &m.Name, &m.URL, &m.CheckInterval, &m.RequestTimeout, &m.ExpectedStatus,
		&m.Status, &m.IsActive, &m.NextCheckAt, &m.LastCheckedAt, &m.LastStatusCode,
		&m.LastResponseMs, &m.ConsecutiveFailures, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update monitor: %w", err)
	}
	return &m, nil
}

// Pause marks a monitor as inactive and sets its status to paused.
func (s *Store) Pause(ctx context.Context, id int64) (*Monitor, error) {
	return s.setActive(ctx, id, false)
}

// Resume marks a monitor as active and schedules an immediate check.
func (s *Store) Resume(ctx context.Context, id int64) (*Monitor, error) {
	return s.setActive(ctx, id, true)
}

func (s *Store) setActive(ctx context.Context, id int64, active bool) (*Monitor, error) {
	status := Status("paused")
	nextCheck := time.Now().UTC().Add(24 * time.Hour * 365) // far future when pausing
	if active {
		status = StatusPending
		nextCheck = time.Now().UTC() // schedule immediately
	}

	var m Monitor
	err := s.db.QueryRow(ctx,
		`UPDATE monitors
		 SET is_active = $1, status = $2, next_check_at = $3, updated_at = NOW()
		 WHERE id = $4
		 RETURNING id, user_id, name, url, check_interval, request_timeout, expected_status,
		           status, is_active, next_check_at, last_checked_at, last_status_code,
		           last_response_ms, consecutive_failures, created_at, updated_at`,
		active, string(status), nextCheck, id,
	).Scan(
		&m.ID, &m.UserID, &m.Name, &m.URL, &m.CheckInterval, &m.RequestTimeout, &m.ExpectedStatus,
		&m.Status, &m.IsActive, &m.NextCheckAt, &m.LastCheckedAt, &m.LastStatusCode,
		&m.LastResponseMs, &m.ConsecutiveFailures, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("set monitor active=%v: %w", active, err)
	}
	return &m, nil
}

// Delete removes a monitor and all associated data (cascaded).
func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM monitors WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DashboardStats returns aggregate counts for a user's monitors.
func (s *Store) DashboardStats(ctx context.Context, userID int64) (*DashboardStats, error) {
	rows, err := s.db.Query(ctx,
		`SELECT status, COUNT(*) FROM monitors WHERE user_id = $1 GROUP BY status`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("dashboard stats: %w", err)
	}
	defer rows.Close()

	stats := &DashboardStats{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stats.Total += count
		switch Status(status) {
		case StatusUp:
			stats.Up = count
		case StatusDown:
			stats.Down = count
		case StatusPaused:
			stats.Paused = count
		case StatusPending:
			stats.Pending = count
		}
	}
	return stats, rows.Err()
}

// DueMonitors returns monitors that are due for checking.
func (s *Store) DueMonitors(ctx context.Context, limit int) ([]*Monitor, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, name, url, check_interval, request_timeout, expected_status,
		        status, is_active, next_check_at, last_checked_at, last_status_code,
		        last_response_ms, consecutive_failures, created_at, updated_at
		 FROM monitors
		 WHERE is_active = TRUE AND next_check_at <= NOW()
		 ORDER BY next_check_at ASC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("due monitors: %w", err)
	}
	defer rows.Close()

	var monitors []*Monitor
	for rows.Next() {
		var m Monitor
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.Name, &m.URL, &m.CheckInterval, &m.RequestTimeout, &m.ExpectedStatus,
			&m.Status, &m.IsActive, &m.NextCheckAt, &m.LastCheckedAt, &m.LastStatusCode,
			&m.LastResponseMs, &m.ConsecutiveFailures, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan due monitor: %w", err)
		}
		monitors = append(monitors, &m)
	}
	return monitors, rows.Err()
}

// UpdateAfterCheck updates monitor state after a check completes.
func (s *Store) UpdateAfterCheck(ctx context.Context, id int64, success bool, statusCode, responseMs int, nextCheckAt time.Time) error {
	newStatus := StatusDown
	if success {
		newStatus = StatusUp
	}

	// Reset consecutive failures on success; increment on failure.
	consecutiveUpdate := "consecutive_failures + 1"
	if success {
		consecutiveUpdate = "0"
	}

	_, err := s.db.Exec(ctx,
		fmt.Sprintf(`UPDATE monitors
		 SET status = $1, last_checked_at = NOW(), last_status_code = $2,
		     last_response_ms = $3, next_check_at = $4,
		     consecutive_failures = %s, updated_at = NOW()
		 WHERE id = $5`, consecutiveUpdate),
		string(newStatus), statusCode, responseMs, nextCheckAt, id,
	)
	if err != nil {
		return fmt.Errorf("update after check: %w", err)
	}
	return nil
}
