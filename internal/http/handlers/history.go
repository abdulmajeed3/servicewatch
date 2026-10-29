package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/amsoft/servicewatch/internal/history"
	mw "github.com/amsoft/servicewatch/internal/http/middleware"
	"github.com/julienschmidt/httprouter"
)

// HistoryHandler serves monitoring check history.
type HistoryHandler struct {
	store *history.Store
}

func NewHistoryHandler(store *history.Store) *HistoryHandler {
	return &HistoryHandler{store: store}
}

// List handles GET /api/v1/monitors/:id/checks.
func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	params := httprouter.ParamsFromContext(r.Context())
	monitorID, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	q := r.URL.Query()
	f := history.ListFilter{
		Page:      parseIntQuery(q.Get("page"), 1),
		Limit:     parseIntQuery(q.Get("limit"), 50),
		OnlyFails: q.Get("only_fails") == "true",
	}

	if v := q.Get("start"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.StartTime = &t
		}
	}
	if v := q.Get("end"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.EndTime = &t
		}
	}

	result, err := h.store.List(r.Context(), monitorID, claims.UserID, f)
	if err != nil {
		mw.WriteError(w, http.StatusInternalServerError, "could not fetch history")
		return
	}

	mw.WriteJSON(w, http.StatusOK, result)
}

// Availability handles GET /api/v1/monitors/:id/availability.
func (h *HistoryHandler) Availability(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	params := httprouter.ParamsFromContext(r.Context())
	monitorID, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	windowHours := parseIntQuery(r.URL.Query().Get("window_hours"), 24)
	if windowHours > 720 {
		windowHours = 720 // cap at 30 days
	}

	stat, err := h.store.Availability(r.Context(), monitorID, claims.UserID, windowHours)
	if err != nil {
		mw.WriteError(w, http.StatusInternalServerError, "could not calculate availability")
		return
	}

	mw.WriteJSON(w, http.StatusOK, stat)
}

// ResponseTimeSeries handles GET /api/v1/monitors/:id/response-times.
func (h *HistoryHandler) ResponseTimeSeries(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	params := httprouter.ParamsFromContext(r.Context())
	monitorID, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	windowHours := parseIntQuery(r.URL.Query().Get("window_hours"), 24)
	since := time.Now().UTC().Add(-time.Duration(windowHours) * time.Hour)

	points, err := h.store.ResponseTimeSeries(r.Context(), monitorID, claims.UserID, since)
	if err != nil {
		mw.WriteError(w, http.StatusInternalServerError, "could not fetch response times")
		return
	}

	mw.WriteJSON(w, http.StatusOK, points)
}
