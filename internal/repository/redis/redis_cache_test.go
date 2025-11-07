package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCache_SetAndGet(t *testing.T) {
	client, _, _ := newTestRedis(t)
	cache := NewCache(client)

	require.NoError(t, cache.Set(ctx, "foo", "bar", time.Minute))
	val, err := cache.Get(ctx, "foo")
	require.NoError(t, err)
	require.Equal(t, "bar", val)
}

func TestCache_GetMiss(t *testing.T) {
	client, _, _ := newTestRedis(t)
	cache := NewCache(client)

	val, err := cache.Get(ctx, "missing")
	require.ErrorIs(t, err, ErrCacheMiss)
	require.Empty(t, val)
}

func TestCache_Delete(t *testing.T) {
	client, _, _ := newTestRedis(t)
	cache := NewCache(client)

	require.NoError(t, cache.Set(ctx, "foo", "bar", time.Minute))
	require.NoError(t, cache.Delete(ctx, "foo"))

	_, err := cache.Get(ctx, "foo")
	require.ErrorIs(t, err, ErrCacheMiss)
}
