package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// PubSub handles Redis-based publish/subscribe fan-out.
type PubSub struct {
	client *redis.Client
}

// NewPubSub constructs a new PubSub instance.
func NewPubSub(client *redis.Client) *PubSub {
	return &PubSub{client: client}
}

// Publish broadcasts payloads to a Redis channel.
func (p *PubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	return p.client.Publish(ctx, channel, payload).Err()
}

// Subscribe listens for messages on a Redis channel.
func (p *PubSub) Subscribe(ctx context.Context, channel string) (<-chan *redis.Message, func() error, error) {
	sub := p.client.Subscribe(ctx, channel)
	if _, err := sub.Receive(ctx); err != nil {
		return nil, nil, err
	}
	return sub.Channel(), sub.Close, nil
}
