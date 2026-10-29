package handlers

import (
	"net/http"

	"github.com/amsoft/servicewatch/internal/history"
	mw "github.com/amsoft/servicewatch/internal/http/middleware"
	"github.com/amsoft/servicewatch/internal/incident"
	"github.com/amsoft/servicewatch/internal/monitor"
)

// DashboardHandler serves aggregated dashboard statistics.
type DashboardHandler struct {
	monitorSvc    *monitor.Service
	historyStore  *history.Store
	incidentStore *incident.Store
}

func NewDashboardHandler(monSvc *monitor.Service, histStore *history.Store, incStore *incident.Store) *DashboardHandler {
	return &DashboardHandler{
		monitorSvc:    monSvc,
		historyStore:  histStore,
		incidentStore: incStore,
	}
}

// Stats handles GET /api/v1/dashboard.
func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID := claims.UserID

	stats, err := h.monitorSvc.DashboardStats(r.Context(), userID)
	if err != nil {
		mw.WriteError(w, http.StatusInternalServerError, "could not fetch stats")
		return
	}

	recentFailures, err := h.historyStore.RecentFailures(r.Context(), userID, 10)
	if err != nil {
		mw.WriteError(w, http.StatusInternalServerError, "could not fetch recent failures")
		return
	}

	recentIncidents, err := h.incidentStore.ListForUser(r.Context(), userID, 5)
	if err != nil {
		mw.WriteError(w, http.StatusInternalServerError, "could not fetch recent incidents")
		return
	}

	type dashboardResponse struct {
		Stats           *monitor.DashboardStats `json:"stats"`
		RecentFailures  any                     `json:"recent_failures"`
		RecentIncidents any                     `json:"recent_incidents"`
	}

	mw.WriteJSON(w, http.StatusOK, dashboardResponse{
		Stats:           stats,
		RecentFailures:  recentFailures,
		RecentIncidents: recentIncidents,
	})
}
