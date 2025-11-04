package http

import "net/http"

// RoomHandler wires room management REST endpoints.
type RoomHandler struct{}

// RegisterRoutes attaches room handlers to the provided mux.
func (h *RoomHandler) RegisterRoutes(mux *http.ServeMux) {
	// TODO: implement room routes
}
