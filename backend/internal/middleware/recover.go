package middleware

import (
	"log"
	"net/http"
)

// Recover catches panics from downstream handlers, logs them, and returns
// a generic 500 instead of letting the process crash or leaking a stack
// trace to the client.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v [%s %s]", err, r.Method, r.URL.Path)
				http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
