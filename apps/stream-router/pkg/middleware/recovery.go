package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery middleware recovers from panics
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the error and stack trace
				log.Printf("PANIC: %v\n%s", err, debug.Stack())

				// Return an error response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				response := map[string]string{
					"error": "Internal server error",
				}

				// Attempt to encode the error
				if encodingErr := json.NewEncoder(w).Encode(response); encodingErr != nil {
					log.Printf("Error encoding panic response: %v", encodingErr)
				}
			}
		}()

		// Process request
		next.ServeHTTP(w, r)
	})
}
