package http

import (
	"net/http"

	"github.com/amsoft/servicewatch/internal/auth"
	"github.com/amsoft/servicewatch/internal/http/handlers"
	mw "github.com/amsoft/servicewatch/internal/http/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"

	"github.com/amsoft/servicewatch/internal/history"
	"github.com/amsoft/servicewatch/internal/incident"
	"github.com/amsoft/servicewatch/internal/monitor"
	"github.com/amsoft/servicewatch/internal/scheduler"
)

// NewRouter assembles the HTTP router with all routes and middleware applied.
func NewRouter(
	db *pgxpool.Pool,
	authSvc *auth.Service,
	monSvc *monitor.Service,
	histStore *history.Store,
	incStore *incident.Store,
	sched *scheduler.Scheduler,
	corsOrigins []string,
) http.Handler {
	authHandler := handlers.NewAuthHandler(authSvc)
	monHandler := handlers.NewMonitorHandler(monSvc, sched)
	histHandler := handlers.NewHistoryHandler(histStore)
	incHandler := handlers.NewIncidentHandler(incStore)
	dashHandler := handlers.NewDashboardHandler(monSvc, histStore, incStore)
	healthHandler := handlers.NewHealthHandler(db)

	router := httprouter.New()

	// Health endpoints (no auth).
	router.HandlerFunc(http.MethodGet, "/health/live", healthHandler.Liveness)
	router.HandlerFunc(http.MethodGet, "/health/ready", healthHandler.Readiness)

	// Auth endpoints (no auth middleware).
	router.HandlerFunc(http.MethodPost, "/api/v1/auth/register", authHandler.Register)
	router.HandlerFunc(http.MethodPost, "/api/v1/auth/login", authHandler.Login)

	// Protected auth endpoints.
	authed := mw.Authenticate(authSvc)
	router.Handler(http.MethodGet, "/api/v1/auth/me", authed(http.HandlerFunc(authHandler.Me)))
	router.Handler(http.MethodPatch, "/api/v1/auth/me", authed(http.HandlerFunc(authHandler.UpdateProfile)))
	router.Handler(http.MethodPost, "/api/v1/auth/change-password", authed(http.HandlerFunc(authHandler.ChangePassword)))

	// Dashboard.
	router.Handler(http.MethodGet, "/api/v1/dashboard", authed(http.HandlerFunc(dashHandler.Stats)))

	// Monitor CRUD.
	router.Handler(http.MethodPost, "/api/v1/monitors", authed(http.HandlerFunc(monHandler.Create)))
	router.Handler(http.MethodGet, "/api/v1/monitors", authed(http.HandlerFunc(monHandler.List)))
	router.Handler(http.MethodGet, "/api/v1/monitors/:id", authed(http.HandlerFunc(monHandler.Get)))
	router.Handler(http.MethodPut, "/api/v1/monitors/:id", authed(http.HandlerFunc(monHandler.Update)))
	router.Handler(http.MethodDelete, "/api/v1/monitors/:id", authed(http.HandlerFunc(monHandler.Delete)))
	router.Handler(http.MethodPost, "/api/v1/monitors/:id/pause", authed(http.HandlerFunc(monHandler.Pause)))
	router.Handler(http.MethodPost, "/api/v1/monitors/:id/resume", authed(http.HandlerFunc(monHandler.Resume)))
	router.Handler(http.MethodPost, "/api/v1/monitors/:id/check", authed(http.HandlerFunc(monHandler.CheckNow)))

	// Monitor check history.
	router.Handler(http.MethodGet, "/api/v1/monitors/:id/checks", authed(http.HandlerFunc(histHandler.List)))
	router.Handler(http.MethodGet, "/api/v1/monitors/:id/availability", authed(http.HandlerFunc(histHandler.Availability)))
	router.Handler(http.MethodGet, "/api/v1/monitors/:id/response-times", authed(http.HandlerFunc(histHandler.ResponseTimeSeries)))

	// Incidents.
	router.Handler(http.MethodGet, "/api/v1/monitors/:id/incidents", authed(http.HandlerFunc(incHandler.List)))

	// Apply global middleware.
	var handler http.Handler = router
	handler = mw.CORS(corsOrigins)(handler)

	return handler
}
