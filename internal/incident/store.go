package incident

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store handles incident persistence.
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// OpenIncident creates a new incident if none is already open for this monitor.
// Returns the incident (new or existing).
func (s *Store) OpenIncident(ctx context.Context, monitorID, userID int64) (*Incident, error) {
	// Check for an existing open incident.
	existing, err := s.GetOpenIncident(ctx, monitorID)
	if err == nil {
		// Increment failure count.
		var inc Incident
		err2 := s.db.QueryRow(ctx,
			`UPDATE incidents
			 SET failure_count = failure_count + 1, last_failure_at = NOW(), updated_at = NOW()
			 WHERE id = $1
			 RETURNING id, monitor_id, user_id, started_at, resolved_at, failure_count, last_failure_at, created_at, updated_at`,
			existing.ID,
		).Scan(&inc.ID, &inc.MonitorID, &inc.UserID, &inc.StartedAt, &inc.ResolvedAt,
			&inc.FailureCount, &inc.LastFailureAt, &inc.CreatedAt, &inc.UpdatedAt)
		if err2 != nil {
			return nil, fmt.Errorf("update incident: %w", err2)
		}
		return &inc, nil
	}
	if !errors.Is(err, ErrNoOpenIncident) {
		return nil, err
	}

	// Create a new incident.
	var inc Incident
	err = s.db.QueryRow(ctx,
		`INSERT INTO incidents (monitor_id, user_id)
		 VALUES ($1, $2)
		 RETURNING id, monitor_id, user_id, started_at, resolved_at, failure_count, last_failure_at, created_at, updated_at`,
		monitorID, userID,
	).Scan(&inc.ID, &inc.MonitorID, &inc.UserID, &inc.StartedAt, &inc.ResolvedAt,
		&inc.FailureCount, &inc.LastFailureAt, &inc.CreatedAt, &inc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("open incident: %w", err)
	}
	return &inc, nil
}

// ResolveIncident closes any open incident for this monitor.
func (s *Store) ResolveIncident(ctx context.Context, monitorID int64) error {
	now := time.Now().UTC()
	tag, err := s.db.Exec(ctx,
		`UPDATE incidents SET resolved_at = $1, updated_at = NOW()
		 WHERE monitor_id = $2 AND resolved_at IS NULL`,
		now, monitorID,
	)
	if err != nil {
		return fmt.Errorf("resolve incident: %w", err)
	}
	_ = tag
	return nil
}

// GetOpenIncident returns the current open incident for a monitor, if any.
func (s *Store) GetOpenIncident(ctx context.Context, monitorID int64) (*Incident, error) {
	var inc Incident
	err := s.db.QueryRow(ctx,
		`SELECT id, monitor_id, user_id, started_at, resolved_at, failure_count, last_failure_at, created_at, updated_at
		 FROM incidents
		 WHERE monitor_id = $1 AND resolved_at IS NULL
		 ORDER BY started_at DESC LIMIT 1`,
		monitorID,
	).Scan(&inc.ID, &inc.MonitorID, &inc.UserID, &inc.StartedAt, &inc.ResolvedAt,
		&inc.FailureCount, &inc.LastFailureAt, &inc.CreatedAt, &inc.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoOpenIncident
	}
	if err != nil {
		return nil, fmt.Errorf("get open incident: %w", err)
	}
	return &inc, nil
}

// List returns a paginated incident list for a monitor.
func (s *Store) List(ctx context.Context, monitorID, userID int64, f ListFilter) (*ListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}
	offset := (f.Page - 1) * f.Limit

	conds := "monitor_id = $1 AND user_id = $2"
	args := []any{monitorID, userID}
	argIdx := 3

	if f.Open != nil {
		if *f.Open {
			conds += " AND resolved_at IS NULL"
		} else {
			conds += " AND resolved_at IS NOT NULL"
		}
	}

	var total int
	if err := s.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM incidents WHERE %s", conds), args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count incidents: %w", err)
	}

	listArgs := append(args, f.Limit, offset)
	rows, err := s.db.Query(ctx,
		fmt.Sprintf(`SELECT id, monitor_id, user_id, started_at, resolved_at, failure_count, last_failure_at, created_at, updated_at
		 FROM incidents WHERE %s
		 ORDER BY started_at DESC
		 LIMIT $%d OFFSET $%d`, conds, argIdx, argIdx+1),
		listArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("list incidents: %w", err)
	}
	defer rows.Close()

	incs := make([]*Incident, 0)
	for rows.Next() {
		var inc Incident
		if err := rows.Scan(&inc.ID, &inc.MonitorID, &inc.UserID, &inc.StartedAt, &inc.ResolvedAt,
			&inc.FailureCount, &inc.LastFailureAt, &inc.CreatedAt, &inc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		incs = append(incs, &inc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate incidents: %w", err)
	}

	return &ListResult{Incidents: incs, TotalCount: total, Page: f.Page, Limit: f.Limit}, nil
}

// ListForUser returns recent incidents across all monitors for a user,
// joined with the monitor name for display purposes.
func (s *Store) ListForUser(ctx context.Context, userID int64, limit int) ([]*Incident, error) {
	rows, err := s.db.Query(ctx,
		`SELECT i.id, i.monitor_id, m.name, i.user_id, i.started_at, i.resolved_at,
		        i.failure_count, i.last_failure_at, i.created_at, i.updated_at
		 FROM incidents i
		 JOIN monitors m ON m.id = i.monitor_id
		 WHERE i.user_id = $1
		 ORDER BY i.started_at DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list user incidents: %w", err)
	}
	defer rows.Close()

	var incs []*Incident
	for rows.Next() {
		var inc Incident
		if err := rows.Scan(&inc.ID, &inc.MonitorID, &inc.MonitorName, &inc.UserID, &inc.StartedAt, &inc.ResolvedAt,
			&inc.FailureCount, &inc.LastFailureAt, &inc.CreatedAt, &inc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		incs = append(incs, &inc)
	}
	return incs, rows.Err()
}

var ErrNoOpenIncident = errors.New("no open incident")
