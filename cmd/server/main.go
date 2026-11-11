package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amsoft/servicewatch/internal/auth"
	"github.com/amsoft/servicewatch/internal/checker"
	"github.com/amsoft/servicewatch/internal/config"
	"github.com/amsoft/servicewatch/internal/history"
	apphttp "github.com/amsoft/servicewatch/internal/http"
	"github.com/amsoft/servicewatch/internal/incident"
	"github.com/amsoft/servicewatch/internal/monitor"
	"github.com/amsoft/servicewatch/internal/scheduler"
	"github.com/amsoft/servicewatch/internal/storage"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := run(log); err != nil {
		log.Error("startup failed", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// ── Database ──────────────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := storage.New(ctx, cfg.Database.DSN,
		cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns, cfg.Database.ConnMaxLifetime)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()
	log.Info("database connected")

	// ── Domain stores and services ────────────────────────────────────────────
	authStore := auth.NewStore(db)
	authSvc := auth.NewService(authStore, cfg.Auth.JWTSecret, cfg.Auth.TokenExpiry, cfg.Auth.BCryptCost, log)

	monStore := monitor.NewStore(db)
	monSvc := monitor.NewService(monStore, log)

	histStore := history.NewStore(db)
	incStore := incident.NewStore(db)

	// ── Monitoring checker ────────────────────────────────────────────────────
	chkr := checker.New(cfg.Monitor.DefaultTimeout, cfg.Monitor.AllowPrivateIPs, cfg.Monitor.MaxResponseBodyRead)

	// ── Scheduler (created before router so it can be injected for on-demand checks) ──
	sched := scheduler.New(monStore, histStore, incStore, chkr,
		cfg.Monitor.WorkerCount, cfg.Monitor.SchedulerInterval, log)

	// ── Router ────────────────────────────────────────────────────────────────
	handler := apphttp.NewRouter(db, authSvc, monSvc, histStore, incStore, sched, cfg.Server.CORSOrigins)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	bgCtx, bgCancel := context.WithCancel(context.Background())

	go func() {
		sched.Run(bgCtx)
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
		}
	}()

	<-stop
	log.Info("shutdown signal received")

	// Stop scheduler.
	bgCancel()

	// Shutdown HTTP server.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Info("server stopped")
	return nil
}
