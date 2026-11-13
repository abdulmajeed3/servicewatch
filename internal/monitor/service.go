package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Service handles monitor business logic.
type Service struct {
	store *Store
	log   *slog.Logger
}

func NewService(store *Store, log *slog.Logger) *Service {
	return &Service{store: store, log: log}
}

// Create validates and creates a new monitor for the given user.
func (s *Service) Create(ctx context.Context, userID int64, req *CreateRequest) (*Monitor, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)

	if req.CheckInterval == 0 {
		req.CheckInterval = 300
	}
	if req.RequestTimeout == 0 {
		req.RequestTimeout = 30
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	m, err := s.store.Create(ctx, userID, req)
	if err != nil {
		return nil, fmt.Errorf("create monitor: %w", err)
	}
	s.log.Info("monitor created", "monitor_id", m.ID, "user_id", userID, "url", m.URL)
	return m, nil
}

// Get returns a monitor owned by the given user.
func (s *Service) Get(ctx context.Context, id, userID int64) (*Monitor, error) {
	return s.store.GetByIDForUser(ctx, id, userID)
}

// List returns a filtered, paginated monitor list for the given user.
func (s *Service) List(ctx context.Context, userID int64, f ListFilter) (*ListResult, error) {
	return s.store.List(ctx, userID, f)
}

// Update modifies a monitor's configuration after validating ownership.
func (s *Service) Update(ctx context.Context, id, userID int64, req *UpdateRequest) (*Monitor, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)

	if err := req.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.store.GetByIDForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	m, err := s.store.Update(ctx, existing.ID, req)
	if err != nil {
		return nil, fmt.Errorf("update monitor: %w", err)
	}
	s.log.Info("monitor updated", "monitor_id", m.ID, "user_id", userID)
	return m, nil
}

// Pause stops checking a monitor.
func (s *Service) Pause(ctx context.Context, id, userID int64) (*Monitor, error) {
	if _, err := s.store.GetByIDForUser(ctx, id, userID); err != nil {
		return nil, err
	}
	m, err := s.store.Pause(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("pause monitor: %w", err)
	}
	s.log.Info("monitor paused", "monitor_id", id, "user_id", userID)
	return m, nil
}

// Resume re-enables checking a paused monitor.
func (s *Service) Resume(ctx context.Context, id, userID int64) (*Monitor, error) {
	if _, err := s.store.GetByIDForUser(ctx, id, userID); err != nil {
		return nil, err
	}
	m, err := s.store.Resume(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("resume monitor: %w", err)
	}
	s.log.Info("monitor resumed", "monitor_id", id, "user_id", userID)
	return m, nil
}

// Delete removes a monitor after verifying ownership.
func (s *Service) Delete(ctx context.Context, id, userID int64) error {
	if _, err := s.store.GetByIDForUser(ctx, id, userID); err != nil {
		return err
	}
	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}
	s.log.Info("monitor deleted", "monitor_id", id, "user_id", userID)
	return nil
}

// Clone duplicates an existing monitor for the same user.
// The new monitor gets "Copy of " prepended to the original name,
// starts with status "pending", and is scheduled for an immediate first check.
func (s *Service) Clone(ctx context.Context, id, userID int64) (*Monitor, error) {
	src, err := s.store.GetByIDForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	name := "Copy of " + src.Name
	if len(name) > 200 {
		name = name[:200]
	}

	req := &CreateRequest{
		Name:           name,
		URL:            src.URL,
		CheckInterval:  src.CheckInterval,
		RequestTimeout: src.RequestTimeout,
		ExpectedStatus: src.ExpectedStatus,
		Notes:          src.Notes,
	}

	m, err := s.store.Create(ctx, userID, req)
	if err != nil {
		return nil, fmt.Errorf("clone monitor: %w", err)
	}
	s.log.Info("monitor cloned", "source_id", id, "new_id", m.ID, "user_id", userID)
	return m, nil
}

// DashboardStats returns aggregate counts for a user's monitors.
func (s *Service) DashboardStats(ctx context.Context, userID int64) (*DashboardStats, error) {
	return s.store.DashboardStats(ctx, userID)
}
