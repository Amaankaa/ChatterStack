package middleware

import (
	"net/http"
	"time"
)

// RateLimiter is a placeholder middleware for request throttling.
func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement rate limiting logic
		time.Sleep(0)
		next.ServeHTTP(w, r)
	})
}
