package websocket

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHubRegisterAndBroadcast(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	client := &Client{
		Send:  make(chan []byte, 1),
		Rooms: []string{"room-1"},
		hub:   hub,
	}

	hub.Register(client)

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()

		_, registered := hub.clients[client]
		members, ok := hub.rooms["room-1"]
		if !registered || !ok {
			return false
		}
		_, inRoom := members[client]
		return inRoom
	}, time.Second, 10*time.Millisecond)

	data := []byte("test-message")
	hub.BroadcastToRoom("room-1", data)

	require.Eventually(t, func() bool {
		select {
		case msg := <-client.Send:
			return bytes.Equal(msg, data)
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}

func TestHubSlowConsumerEvicted(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	client := &Client{
		Send:  make(chan []byte),
		Rooms: []string{"room-1"},
		hub:   hub,
	}

	hub.Register(client)

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		_, registered := hub.clients[client]
		hub.mu.RUnlock()
		return registered
	}, time.Second, 10*time.Millisecond)

	hub.BroadcastToRoom("room-1", []byte("payload"))

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		_, registered := hub.clients[client]
		return !registered
	}, time.Second, 10*time.Millisecond)
}

func TestHubShutdownClosesClientChannels(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	go hub.Run(ctx)

	client := &Client{
		Send:  make(chan []byte, 1),
		Rooms: []string{"room-1"},
		hub:   hub,
	}

	hub.Register(client)

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		_, registered := hub.clients[client]
		hub.mu.RUnlock()
		return registered
	}, time.Second, 10*time.Millisecond)

	done := make(chan struct{})
	go func() {
		_, ok := <-client.Send
		if !ok {
			close(done)
		}
	}()

	cancel()

	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}
