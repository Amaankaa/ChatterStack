package http

import "net/http"

// MessageHandler wires REST endpoints for message retrieval and creation.
type MessageHandler struct{}

// RegisterRoutes attaches message handlers to the provided mux.
func (h *MessageHandler) RegisterRoutes(mux *http.ServeMux) {
	// TODO: implement message routes
}
