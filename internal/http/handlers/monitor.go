package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	mw "github.com/amsoft/servicewatch/internal/http/middleware"
	"github.com/amsoft/servicewatch/internal/monitor"
	"github.com/julienschmidt/httprouter"
)

// MonitorHandler handles monitor management HTTP requests.
type MonitorHandler struct {
	svc *monitor.Service
}

func NewMonitorHandler(svc *monitor.Service) *MonitorHandler {
	return &MonitorHandler{svc: svc}
}

// Create handles POST /api/v1/monitors.
func (h *MonitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req monitor.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	m, err := h.svc.Create(r.Context(), claims.UserID, &req)
	if err != nil {
		handleMonitorError(w, err)
		return
	}

	mw.WriteJSON(w, http.StatusCreated, m)
}

// List handles GET /api/v1/monitors.
func (h *MonitorHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	q := r.URL.Query()
	f := monitor.ListFilter{
		Search: q.Get("search"),
		Status: monitor.Status(q.Get("status")),
		Page:   parseIntQuery(q.Get("page"), 1),
		Limit:  parseIntQuery(q.Get("limit"), 20),
	}

	result, err := h.svc.List(r.Context(), claims.UserID, f)
	if err != nil {
		mw.WriteError(w, http.StatusInternalServerError, "could not list monitors")
		return
	}

	mw.WriteJSON(w, http.StatusOK, result)
}

// Get handles GET /api/v1/monitors/:id.
func (h *MonitorHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	m, err := h.svc.Get(r.Context(), id, claims.UserID)
	if err != nil {
		handleMonitorError(w, err)
		return
	}

	mw.WriteJSON(w, http.StatusOK, m)
}

// Update handles PUT /api/v1/monitors/:id.
func (h *MonitorHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	var req monitor.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	m, err := h.svc.Update(r.Context(), id, claims.UserID, &req)
	if err != nil {
		handleMonitorError(w, err)
		return
	}

	mw.WriteJSON(w, http.StatusOK, m)
}

// Pause handles POST /api/v1/monitors/:id/pause.
func (h *MonitorHandler) Pause(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	m, err := h.svc.Pause(r.Context(), id, claims.UserID)
	if err != nil {
		handleMonitorError(w, err)
		return
	}

	mw.WriteJSON(w, http.StatusOK, m)
}

// Resume handles POST /api/v1/monitors/:id/resume.
func (h *MonitorHandler) Resume(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	m, err := h.svc.Resume(r.Context(), id, claims.UserID)
	if err != nil {
		handleMonitorError(w, err)
		return
	}

	mw.WriteJSON(w, http.StatusOK, m)
}

// Delete handles DELETE /api/v1/monitors/:id.
func (h *MonitorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	if err := h.svc.Delete(r.Context(), id, claims.UserID); err != nil {
		handleMonitorError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleMonitorError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, monitor.ErrNotFound):
		mw.WriteError(w, http.StatusNotFound, "monitor not found")
	case errors.Is(err, monitor.ErrForbidden):
		mw.WriteError(w, http.StatusForbidden, "access denied")
	case errors.Is(err, monitor.ErrInvalidInput):
		mw.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		mw.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func parseIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())
	return strconv.ParseInt(params.ByName("id"), 10, 64)
}

func parseIntQuery(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return def
	}
	return v
}
