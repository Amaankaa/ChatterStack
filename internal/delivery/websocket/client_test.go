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

	if s.hub != nil {
		event, err := Event{Type: EventReceiveMessage, Data: receiveMessagePayload{
			ID:          msg.ID,
			RoomID:      msg.RoomID,
			SenderID:    msg.SenderID,
			Content:     msg.Content,
			Attachments: toPayloadAttachments(msg.Attachments),
			Status:      msg.Status,
			CreatedAt:   msg.CreatedAt,
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
	client := NewClient(nil, hub, "user-1", []string{"room-1"}, sender)

	require.Equal(t, hub, client.hub)
	require.Equal(t, "user-1", client.UserID)
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

		client := NewClient(conn, hub, "user-1", []string{"room-1"}, sender)
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

	require.Equal(t, "room-1", sender.last.RoomID)
	require.Equal(t, "user-1", sender.last.SenderID)
	require.Equal(t, "hello world", sender.last.Content)
}
