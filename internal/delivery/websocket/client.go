package websocket

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a connected websocket participant.
type Client struct {
	Conn   *websocket.Conn
	Send   chan []byte
	UserID string
	Rooms  []string

	hub *Hub
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 5120
)

// NewClient wraps a websocket connection and attaches it to the hub.
func NewClient(conn *websocket.Conn, hub *Hub, userID string, rooms []string) *Client {
	return &Client{
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
		Rooms:  rooms,
		hub:    hub,
	}
}

// ReadPump pumps inbound messages from the websocket connection.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket: read error: %v", err)
			}
			break
		}

		//For now, treat every inbound payload as a broadcast to the client's rooms.
		for _, room := range c.Rooms {
			c.hub.BroadcastToRoom(room, message)
		}
	}
}

// WritePump pumps outbound messages to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				_ = w.Close()
				return
			}

			//Flush any queued messages to reduce frame count.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				if _, err := w.Write([]byte{'\n'}); err != nil {
					break
				}
				if _, err := w.Write(<-c.Send); err != nil {
					break
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
