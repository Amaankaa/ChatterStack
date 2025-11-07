package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPubSub_PublishSubscribe(t *testing.T) {
	client, _, _ := newTestRedis(t)
	ps := NewPubSub(client)

	ch, cancel, err := ps.Subscribe(ctx, "chat-room-1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = cancel() })

	require.NoError(t, ps.Publish(ctx, "chat-room-1", []byte("hello")))

	select {
	case msg := <-ch:
		require.Equal(t, "chat-room-1", msg.Channel)
		require.Equal(t, "hello", msg.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}
