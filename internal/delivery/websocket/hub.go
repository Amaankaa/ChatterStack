package websocket

// Hub orchestrates websocket clients and room broadcasts.
type Hub struct {
	// TODO: add client registries, broadcast channels, etc.
}

// NewHub constructs an empty Hub instance.
func NewHub() *Hub {
	return &Hub{}
}

// Run starts the hub event loop.
func (h *Hub) Run() {
	// TODO: implement hub event loop
}
