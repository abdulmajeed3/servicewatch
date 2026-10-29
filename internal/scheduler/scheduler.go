// Package scheduler implements a bounded worker pool that periodically finds
// monitors due for checking and dispatches them to workers.
//
// Design
// ──────
// A single ticker fires every SchedulerInterval. On each tick it queries
// monitors WHERE is_active = TRUE AND next_check_at <= NOW(). Each monitor is
// dispatched to a semaphore-bounded goroutine. The semaphore ensures at most
// WorkerCount checks run concurrently.
//
// A monitor is not re-queued until its worker completes and writes next_check_at
// back to the database, so the same monitor cannot be checked concurrently by
// two workers (absent a multi-process deployment, which this design does not
// support).
//
// Graceful shutdown: the run loop exits when ctx is cancelled. In-flight
// workers finish their current check (up to the per-check timeout), then write
// results before returning.
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/amsoft/servicewatch/internal/checker"
	"github.com/amsoft/servicewatch/internal/history"
	"github.com/amsoft/servicewatch/internal/incident"
	"github.com/amsoft/servicewatch/internal/monitor"
)

// Scheduler orchestrates periodic monitor checking.
type Scheduler struct {
	monitorStore  *monitor.Store
	historyStore  *history.Store
	incidentStore *incident.Store
	checker       *checker.Checker
	workerCount   int
	tickInterval  time.Duration
	log           *slog.Logger
}

// New creates a Scheduler. Call Run to start it.
func New(
	monitorStore *monitor.Store,
	historyStore *history.Store,
	incidentStore *incident.Store,
	chkr *checker.Checker,
	workerCount int,
	tickInterval time.Duration,
	log *slog.Logger,
) *Scheduler {
	if workerCount < 1 {
		workerCount = 10
	}
	return &Scheduler{
		monitorStore:  monitorStore,
		historyStore:  historyStore,
		incidentStore: incidentStore,
		checker:       chkr,
		workerCount:   workerCount,
		tickInterval:  tickInterval,
		log:           log,
	}
}

// Run starts the scheduler loop. It blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	s.log.Info("scheduler started", "workers", s.workerCount, "interval", s.tickInterval)
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	// Semaphore to bound concurrent workers.
	sem := make(chan struct{}, s.workerCount)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			s.log.Info("scheduler shutting down — waiting for in-flight checks")
			wg.Wait()
			s.log.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.dispatch(ctx, sem, &wg)
		}
	}
}

func (s *Scheduler) dispatch(ctx context.Context, sem chan struct{}, wg *sync.WaitGroup) {
	monitors, err := s.monitorStore.DueMonitors(ctx, s.workerCount*2)
	if err != nil {
		s.log.Error("fetch due monitors", "err", err)
		return
	}
	if len(monitors) == 0 {
		return
	}

	s.log.Debug("dispatching checks", "count", len(monitors))

	for _, m := range monitors {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(m *monitor.Monitor) {
			defer wg.Done()
			defer func() { <-sem }()
			// Use a background context for the check so shutdown cancellation
			// does not abandon database writes; the per-check timeout is applied inside Check().
			s.runCheck(context.Background(), m)
		}(m)
	}
}

func (s *Scheduler) runCheck(ctx context.Context, m *monitor.Monitor) {
	timeout := time.Duration(m.RequestTimeout) * time.Second
	result := s.checker.Check(ctx, m.URL, m.ExpectedStatus, timeout)

	nextCheckAt := result.CheckedAt.Add(time.Duration(m.CheckInterval) * time.Second)

	// Persist check result.
	if err := s.historyStore.Record(ctx, m.ID, m.UserID,
		result.IsSuccess, result.StatusCode, result.ResponseMs, result.FailureReason); err != nil {
		s.log.Error("record check", "monitor_id", m.ID, "err", err)
	}

	// Update monitor status.
	if err := s.monitorStore.UpdateAfterCheck(ctx, m.ID,
		result.IsSuccess, result.StatusCode, result.ResponseMs, nextCheckAt); err != nil {
		s.log.Error("update after check", "monitor_id", m.ID, "err", err)
	}

	// Manage incidents.
	if !result.IsSuccess {
		if _, err := s.incidentStore.OpenIncident(ctx, m.ID, m.UserID); err != nil {
			s.log.Error("open incident", "monitor_id", m.ID, "err", err)
		}
	} else {
		if err := s.incidentStore.ResolveIncident(ctx, m.ID); err != nil {
			s.log.Error("resolve incident", "monitor_id", m.ID, "err", err)
		}
	}

	s.log.Debug("check complete",
		"monitor_id", m.ID,
		"url", m.URL,
		"success", result.IsSuccess,
		"status_code", result.StatusCode,
		"response_ms", result.ResponseMs,
		"failure", result.FailureReason,
	)
}
