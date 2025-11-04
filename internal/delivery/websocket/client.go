package websocket

// Client represents a connected websocket participant.
type Client struct {
	// TODO: add underlying connection implementation
	Send   chan []byte
	UserID string
}

// ReadPump pumps inbound messages from the websocket connection.
func (c *Client) ReadPump(h *Hub) {
	// TODO: implement read loop
}

// WritePump pumps outbound messages to the websocket connection.
func (c *Client) WritePump() {
	// TODO: implement write loop
}
