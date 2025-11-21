package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig controls the behaviour of the rate limiting middleware.
type RateLimiterConfig struct {
	Requests int
	Window   time.Duration
	Burst    int
	KeyFunc  func(*gin.Context) string
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	limit    rate.Limit
	burst    int
	window   time.Duration
	keyFunc  func(*gin.Context) string
	mu       sync.Mutex
	visitors map[string]*visitor
}

// RateLimit creates a Gin middleware that throttles requests per client key.
func RateLimit(cfg RateLimiterConfig) gin.HandlerFunc {
	if cfg.Requests <= 0 {
		cfg.Requests = 100
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if cfg.Burst <= 0 {
		cfg.Burst = cfg.Requests
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *gin.Context) string { return c.ClientIP() }
	}

	rl := &rateLimiter{
		limit:    rate.Limit(float64(cfg.Requests) / cfg.Window.Seconds()),
		burst:    cfg.Burst,
		window:   cfg.Window,
		keyFunc:  cfg.KeyFunc,
		visitors: make(map[string]*visitor),
	}

	go rl.cleanup()

	return rl.handle
}

func (r *rateLimiter) handle(c *gin.Context) {
	key := r.keyFunc(c)
	limiter := r.getVisitor(key)
	if limiter.Allow() {
		c.Next()
		return
	}

	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
}

func (r *rateLimiter) getVisitor(key string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok := r.visitors[key]; ok {
		v.lastSeen = time.Now()
		return v.limiter
	}

	limiter := rate.NewLimiter(r.limit, r.burst)
	r.visitors[key] = &visitor{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

func (r *rateLimiter) cleanup() {
	ticker := time.NewTicker(r.window)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		cutoff := time.Now().Add(-r.window * 2)
		for key, v := range r.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(r.visitors, key)
			}
		}
		r.mu.Unlock()
	}
}
