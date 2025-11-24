package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chatterstack/internal/domain/messages"
	"chatterstack/internal/domain/models"

	ws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type stubMessageSender struct {
	hub  *Hub
	last messages.SendMessageInput
}

func newStubMessageSender(hub *Hub) *stubMessageSender {
	return &stubMessageSender{hub: hub}
}

func (s *stubMessageSender) Send(_ context.Context, input messages.SendMessageInput) (*models.Message, error) {
	s.last = input

	msg := &models.Message{
		ID:          "msg-1",
		RoomID:      input.RoomID,
		SenderID:    input.SenderID,
		Content:     input.Content,
		Attachments: input.Attachments,
		Status:      models.MessageStatusSent,
		CreatedAt:   time.Now().UTC(),
	}
	msg.UpdatedAt = msg.CreatedAt

	if s.hub != nil {
		event, err := Event{Type: EventReceiveMessage, Data: receiveMessagePayload{
			ID:          msg.ID,
			RoomID:      msg.RoomID,
			SenderID:    msg.SenderID,
			Content:     msg.Content,
			Attachments: toPayloadAttachments(msg.Attachments),
			Status:      msg.Status,
			CreatedAt:   msg.CreatedAt,
			UpdatedAt:   msg.UpdatedAt,
		}}.Encode()
		if err == nil {
			s.hub.BroadcastToRoom(msg.RoomID, event)
		}
	}

	return msg, nil
}

func TestNewClientInitialisation(t *testing.T) {
	t.Helper()

	hub := NewHub()
	sender := newStubMessageSender(nil)
	client := NewClient(nil, hub, "user-1", []string{"room-1"}, sender, "alice")

	require.Equal(t, hub, client.hub)
	require.Equal(t, "user-1", client.UserID)
	require.Equal(t, "alice", client.Username)
	require.Equal(t, []string{"room-1"}, client.Rooms)
	require.NotNil(t, client.Send)
	require.Equal(t, 256, cap(client.Send))
	require.Equal(t, sender, client.sender)
}

func TestClientReadPumpBroadcastsAndEchoes(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	upgrader := ws.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	sender := newStubMessageSender(hub)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade error: %v", err)
			return
		}

		client := NewClient(conn, hub, "user-1", []string{"room-1"}, sender, "alice")
		hub.Register(client)

		go client.WritePump()
		client.ReadPump(ctx)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	outbound, err := Event{Type: EventSendMessage, Data: sendMessagePayload{RoomID: "room-1", Content: "hello world"}}.Encode()
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(ws.TextMessage, outbound))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	require.NoError(t, err)

	if idx := bytes.IndexByte(raw, '\n'); idx >= 0 {
		raw = raw[:idx]
	}

	var envelope struct {
		Event EventType              `json:"event"`
		Data  map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &envelope))
	require.Equal(t, EventReceiveMessage, envelope.Event)
	require.Equal(t, "hello world", envelope.Data["content"])
	require.Equal(t, "room-1", envelope.Data["room_id"])
	require.Equal(t, "user-1", envelope.Data["sender_id"])
	require.NotEmpty(t, envelope.Data["updated_at"])

	require.Equal(t, "room-1", sender.last.RoomID)
	require.Equal(t, "user-1", sender.last.SenderID)
	require.Equal(t, "hello world", sender.last.Content)
}

func TestClientTypingEventsBroadcastToOthers(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	clientA := NewClient(nil, hub, "user-1", []string{"room-1"}, nil, "Alice")
	clientB := NewClient(nil, hub, "user-2", []string{"room-1"}, nil, "Bob")

	hub.Register(clientA)
	hub.Register(clientB)

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.rooms["room-1"]) == 2
	}, time.Second, 10*time.Millisecond)

	clientA.handleTypingEvent(EventTypingStart, typingEventPayload{RoomID: "room-1"})

	select {
	case raw := <-clientB.Send:
		var envelope struct {
			Event EventType         `json:"event"`
			Data  map[string]string `json:"data"`
		}
		require.NoError(t, json.Unmarshal(raw, &envelope))
		require.Equal(t, EventTypingStart, envelope.Event)
		require.Equal(t, "Alice", envelope.Data["username"])
		require.Equal(t, "room-1", envelope.Data["room_id"])
	case <-time.After(time.Second):
		t.Fatal("expected typing event for clientB")
	}

	select {
	case <-clientA.Send:
		t.Fatal("did not expect sender to receive typing event")
	default:
	}
}
