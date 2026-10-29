package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/amsoft/servicewatch/internal/auth"
	mw "github.com/amsoft/servicewatch/internal/http/middleware"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	svc *auth.Service
}

func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	resp, err := h.svc.Register(r.Context(), &req)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	mw.WriteJSON(w, http.StatusCreated, resp)
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	resp, err := h.svc.Login(r.Context(), &req)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	mw.WriteJSON(w, http.StatusOK, resp)
}

// Me handles GET /api/v1/auth/me.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.svc.GetUser(r.Context(), claims.UserID)
	if err != nil {
		mw.WriteError(w, http.StatusInternalServerError, "could not fetch user")
		return
	}

	mw.WriteJSON(w, http.StatusOK, user)
}

// UpdateProfile handles PATCH /api/v1/auth/me.
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req auth.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := h.svc.UpdateProfile(r.Context(), claims.UserID, &req)
	if err != nil {
		mw.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	mw.WriteJSON(w, http.StatusOK, user)
}

// ChangePassword handles POST /api/v1/auth/change-password.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFromContext(r.Context())
	if !ok {
		mw.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req auth.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mw.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := h.svc.ChangePassword(r.Context(), claims.UserID, &req); err != nil {
		handleAuthError(w, err)
		return
	}

	mw.WriteJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

func handleAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		mw.WriteError(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, auth.ErrEmailAlreadyExists):
		mw.WriteError(w, http.StatusConflict, "email already registered")
	case errors.Is(err, auth.ErrUserNotFound):
		mw.WriteError(w, http.StatusNotFound, "user not found")
	default:
		mw.WriteError(w, http.StatusBadRequest, err.Error())
	}
}
