package handlers

import (
	"context"
	"net/http"
	"time"

	mw "github.com/amsoft/servicewatch/internal/http/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler provides liveness and readiness probes.
type HealthHandler struct {
	db *pgxpool.Pool
}

func NewHealthHandler(db *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{db: db}
}

// Liveness handles GET /health/live — always 200 if the process is running.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	mw.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readiness handles GET /health/ready — checks database connectivity.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		mw.WriteError(w, http.StatusServiceUnavailable, "database not ready")
		return
	}

	mw.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
