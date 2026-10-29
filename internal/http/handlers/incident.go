package handlers

import (
	"net/http"
	"strconv"

	mw "github.com/amsoft/servicewatch/internal/http/middleware"
	"github.com/amsoft/servicewatch/internal/incident"
	"github.com/julienschmidt/httprouter"
)

// IncidentHandler serves incident data.
type IncidentHandler struct {
	store *incident.Store
}

func NewIncidentHandler(store *incident.Store) *IncidentHandler {
	return &IncidentHandler{store: store}
}

// List handles GET /api/v1/monitors/:id/incidents.
func (h *IncidentHandler) List(w http.ResponseWriter, r *http.Request) {
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
	f := incident.ListFilter{
		Page:  parseIntQuery(q.Get("page"), 1),
		Limit: parseIntQuery(q.Get("limit"), 20),
	}
	if v := q.Get("open"); v == "true" {
		b := true
		f.Open = &b
	} else if v == "false" {
		b := false
		f.Open = &b
	}

	result, err := h.store.List(r.Context(), monitorID, claims.UserID, f)
	if err != nil {
		mw.WriteError(w, http.StatusInternalServerError, "could not fetch incidents")
		return
	}

	mw.WriteJSON(w, http.StatusOK, result)
}
