package http

import (
	"net/http"
)

// AuthHandler wires authentication HTTP endpoints.
type AuthHandler struct{}

// RegisterRoutes attaches authentication routes to the provided mux.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	// TODO: implement auth routes
}
