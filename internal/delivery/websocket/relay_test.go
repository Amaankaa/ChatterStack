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

func (f *fakePatternSubscriber) publish(payload []byte) {
	f.ch <- &redis.Message{Payload: string(payload)}
}

func TestStartMessageRelayBroadcastsMessages(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	go hub.Run(ctx)

	subscriber := newFakePatternSubscriber()
	StartMessageRelay(ctx, hub, subscriber)

	require.Eventually(t, func() bool { return len(subscriber.patterns) == 1 && subscriber.patterns[0][0] == roomMessagePattern }, time.Second, 10*time.Millisecond)

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

	subscriber.publish(payload)

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

	subscriber.publish(payload)

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
