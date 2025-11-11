package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestNewClientInitialisation(t *testing.T) {
	t.Helper()

	hub := NewHub()
	client := NewClient(nil, hub, "user-1", []string{"room-1"})

	require.Equal(t, hub, client.hub)
	require.Equal(t, "user-1", client.UserID)
	require.Equal(t, []string{"room-1"}, client.Rooms)
	require.NotNil(t, client.Send)
	require.Equal(t, 256, cap(client.Send))
}

func TestClientReadPumpBroadcastsAndEchoes(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	upgrader := ws.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade error: %v", err)
			return
		}

		client := NewClient(conn, hub, "user-1", []string{"room-1"})
		hub.Register(client)

		go client.WritePump()
		client.ReadPump()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	payload := []byte("hello world")
	require.NoError(t, conn.WriteMessage(ws.TextMessage, payload))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, echo, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, payload, echo)
}
