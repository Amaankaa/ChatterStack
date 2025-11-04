package http

import "net/http"

// UserHandler exposes REST endpoints for user-specific operations.
type UserHandler struct{}

// RegisterRoutes attaches user handlers to the provided mux.
func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	// TODO: implement user routes
}
