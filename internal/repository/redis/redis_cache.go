package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache provides access to Redis-based caching primitives.
type Cache struct {
	client *redis.Client
}

// NewCache constructs a new Cache instance.
func NewCache(client *redis.Client) *Cache {
	return &Cache{client: client}
}

// Get fetches a cached value by key.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrCacheMiss
	}
	return val, err
}

// Set stores a value with an optional TTL.
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
