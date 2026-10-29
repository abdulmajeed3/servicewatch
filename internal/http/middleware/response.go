package middleware

import (
	"encoding/json"
	"net/http"
)

// APIResponse is the standard JSON envelope for all API responses.
type APIResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// WriteJSON writes a JSON response with the given status code and data.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{Data: data})
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{Error: message})
}

// WriteValidationErrors writes a 422 response with field errors.
func WriteValidationErrors(w http.ResponseWriter, errs map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	type validationError struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	_ = json.NewEncoder(w).Encode(validationError{
		Error:  "validation failed",
		Fields: errs,
	})
}
