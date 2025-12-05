package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"chatterstack/internal/domain/models"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type fakePatternSubscriber struct {
	ch       chan *redis.Message
	patterns [][]string
	closed   chan struct{}
	once     sync.Once
}

func newFakePatternSubscriber() *fakePatternSubscriber {
	return &fakePatternSubscriber{
		ch:     make(chan *redis.Message, 1),
		closed: make(chan struct{}),
	}
}

func (f *fakePatternSubscriber) PatternSubscribe(ctx context.Context, patterns ...string) (<-chan *redis.Message, func() error, error) {
	copyPatterns := append([]string(nil), patterns...)
	f.patterns = append(f.patterns, copyPatterns)
	return f.ch, func() error {
		f.once.Do(func() { close(f.closed) })
		return nil
	}, nil
}

func (f *fakePatternSubscriber) publish(channel string, payload []byte) {
	f.ch <- &redis.Message{Channel: channel, Payload: string(payload)}
}

func TestStartMessageRelayBroadcastsMessages(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	go hub.Run(ctx)

	subscriber := newFakePatternSubscriber()
	StartMessageRelay(ctx, hub, subscriber)

	require.Eventually(t, func() bool {
		if len(subscriber.patterns) != 1 {
			return false
		}
		patterns := subscriber.patterns[0]
		hasRoom := false
		hasUser := false
		for _, p := range patterns {
			if p == roomMessagePattern {
				hasRoom = true
			}
			if p == userEventPattern {
				hasUser = true
			}
		}
		return hasRoom && hasUser
	}, time.Second, 10*time.Millisecond)

	client := &Client{
		Send:    make(chan []byte, 1),
		Rooms:   []string{"room-1"},
		roomSet: map[string]struct{}{"room-1": {}},
	}
	hub.Register(client)

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.rooms["room-1"]) == 1
	}, time.Second, 10*time.Millisecond)

	msg := models.Message{
		ID:        "msg-1",
		RoomID:    "room-1",
		SenderID:  "user-1",
		Content:   "hi",
		Status:    models.MessageStatusSent,
		CreatedAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(msg)
	require.NoError(t, err)

	subscriber.publish("rooms:room-1:messages", payload)

	select {
	case data := <-client.Send:
		var envelope struct {
			Event EventType              `json:"event"`
			Data  map[string]interface{} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(data, &envelope))
		require.Equal(t, EventReceiveMessage, envelope.Event)
		require.Equal(t, "room-1", envelope.Data["room_id"])
		require.Equal(t, "user-1", envelope.Data["sender_id"])
		require.Equal(t, "hi", envelope.Data["content"])
		_, ok := envelope.Data["updated_at"]
		require.True(t, ok, "expected updated_at in payload")
	case <-time.After(time.Second):
		t.Fatal("expected broadcast message")
	}

	cancel()

	select {
	case <-subscriber.closed:
	case <-time.After(time.Second):
		t.Fatal("expected subscriber close")
	}
}

func TestStartMessageRelayBroadcastsDeleteEvents(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	subscriber := newFakePatternSubscriber()
	StartMessageRelay(ctx, hub, subscriber)

	require.Eventually(t, func() bool { return len(subscriber.patterns) == 1 }, time.Second, 10*time.Millisecond)

	client := &Client{
		Send:    make(chan []byte, 1),
		Rooms:   []string{"room-1"},
		roomSet: map[string]struct{}{"room-1": {}},
	}
	hub.Register(client)

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.rooms["room-1"]) == 1
	}, time.Second, 10*time.Millisecond)

	payload, err := json.Marshal(map[string]any{
		"type": "message.deleted",
		"message": map[string]any{
			"id":      "msg-1",
			"room_id": "room-1",
		},
	})
	require.NoError(t, err)

	subscriber.publish("rooms:room-1:messages", payload)

	select {
	case data := <-client.Send:
		var envelope struct {
			Event EventType         `json:"event"`
			Data  map[string]string `json:"data"`
		}
		require.NoError(t, json.Unmarshal(data, &envelope))
		require.Equal(t, EventMessageDeleted, envelope.Event)
		require.Equal(t, "msg-1", envelope.Data["id"])
		require.Equal(t, "room-1", envelope.Data["room_id"])
	case <-time.After(time.Second):
		t.Fatal("expected delete broadcast message")
	}
}

func TestStartMessageRelayBroadcastsUserEvents(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	subscriber := newFakePatternSubscriber()
	StartMessageRelay(ctx, hub, subscriber)

	require.Eventually(t, func() bool { return len(subscriber.patterns) == 1 }, time.Second, 10*time.Millisecond)

	client := &Client{
		Send:     make(chan []byte, 1),
		UserID:   "user-1",
		Username: "User 1",
		Rooms:    []string{},
		roomSet:  map[string]struct{}{},
	}
	hub.Register(client)

	// Wait for client to be registered in user map
	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.users["user-1"]) == 1
	}, time.Second, 10*time.Millisecond)

	payload := map[string]interface{}{
		"event": "room_added",
		"data": map[string]interface{}{
			"id":   "room-new",
			"name": "New Room",
		},
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	subscriber.publish("users:user-1:events", payloadBytes)

	select {
	case data := <-client.Send:
		var envelope map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &envelope))
		require.Equal(t, "room_added", envelope["event"])
		dataMap := envelope["data"].(map[string]interface{})
		require.Equal(t, "room-new", dataMap["id"])
		require.Equal(t, "New Room", dataMap["name"])
	case <-time.After(time.Second):
		t.Fatal("expected broadcast message")
	}
}
