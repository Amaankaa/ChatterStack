package redis

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t testing.TB) (*redis.Client, *miniredis.Miniredis, func()) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	cleanup := func() {
		client.Close()
		srv.Close()
	}
	t.Cleanup(cleanup)
	return client, srv, cleanup
}

var ctx = context.Background()
