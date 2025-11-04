package response

import "net/http"

// JSON writes a JSON response with the provided status code.
func JSON(w http.ResponseWriter, status int, payload interface{}) {
	// TODO: marshal payload and write to response writer
	w.WriteHeader(status)
}
