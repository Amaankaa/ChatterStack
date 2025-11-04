package middleware

import "net/http"

// AuthMiddleware verifies JWT tokens before passing requests downstream.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement JWT verification
		next.ServeHTTP(w, r)
	})
}
