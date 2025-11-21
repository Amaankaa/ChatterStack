package websocket

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"chatterstack/internal/domain/messages"
	"chatterstack/internal/domain/models"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

// MessageSender abstracts the message use case for inbound websocket events.
type MessageSender interface {
	Send(ctx context.Context, input messages.SendMessageInput) (*models.Message, error)
}

// Client represents a connected websocket participant.
type Client struct {
	Conn    *websocket.Conn
	Send    chan []byte
	UserID  string
	Rooms   []string
	limiter *rate.Limiter

	hub     *Hub
	sender  MessageSender
	roomSet map[string]struct{}
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 5120
)

// NewClient wraps a websocket connection and attaches it to the hub.
func NewClient(conn *websocket.Conn, hub *Hub, userID string, rooms []string, sender MessageSender) *Client {
	roomSet := make(map[string]struct{}, len(rooms))
	for _, room := range rooms {
		roomSet[room] = struct{}{}
	}

	return &Client{
		Conn:    conn,
		Send:    make(chan []byte, 256),
		UserID:  userID,
		Rooms:   rooms,
		limiter: rate.NewLimiter(rate.Every(200*time.Millisecond), 5),
		hub:     hub,
		sender:  sender,
		roomSet: roomSet,
	}
}

// ReadPump pumps inbound messages from the websocket connection.
func (c *Client) ReadPump(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

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

		if c.limiter != nil && !c.limiter.Allow() {
			log.Printf("websocket: rate limit exceeded for user %s", c.UserID)
			continue
		}

		var envelope struct {
			Event EventType       `json:"event"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(message, &envelope); err != nil {
			log.Printf("websocket: invalid payload from %s: %v", c.UserID, err)
			continue
		}

		switch envelope.Event {
		case EventSendMessage:
			var payload sendMessagePayload
			if err := json.Unmarshal(envelope.Data, &payload); err != nil {
				log.Printf("websocket: malformed send_message payload: %v", err)
				continue
			}
			c.handleSendMessage(ctx, payload)
		default:
			log.Printf("websocket: unsupported event %q for user %s", envelope.Event, c.UserID)
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

func (c *Client) handleSendMessage(ctx context.Context, payload sendMessagePayload) {
	if c.sender == nil {
		log.Printf("websocket: message sender missing for user %s", c.UserID)
		return
	}

	roomID := strings.TrimSpace(payload.RoomID)
	if roomID == "" {
		log.Printf("websocket: missing room_id in send_message event for user %s", c.UserID)
		return
	}
	if !c.isMember(roomID) {
		log.Printf("websocket: user %s attempted to publish to room %s without membership", c.UserID, roomID)
		return
	}

	msg, err := c.sender.Send(ctx, messages.SendMessageInput{
		RoomID:      roomID,
		SenderID:    c.UserID,
		Content:     strings.TrimSpace(payload.Content),
		Attachments: toDomainAttachments(payload.Attachments),
	})
	if err != nil {
		log.Printf("websocket: send message failed: %v", err)
		return
	}

	eventPayload, err := Event{Type: EventReceiveMessage, Data: receiveMessagePayload{
		ID:          msg.ID,
		RoomID:      msg.RoomID,
		SenderID:    msg.SenderID,
		Content:     msg.Content,
		Attachments: toPayloadAttachments(msg.Attachments),
		Status:      msg.Status,
		CreatedAt:   msg.CreatedAt,
	}}.Encode()
	if err != nil {
		log.Printf("websocket: encode receive_message event failed: %v", err)
		return
	}

	c.hub.BroadcastToRoom(msg.RoomID, eventPayload)
}

func (c *Client) isMember(roomID string) bool {
	_, ok := c.roomSet[roomID]
	return ok
}
