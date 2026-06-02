package api

import (
	"context"
	"log"
	"net/http"
	"time"
)

// withNode returns a request whose context carries the authenticated node ID.
func withNode(r *http.Request, nodeID string) context.Context {
	return context.WithValue(r.Context(), nodeIDKey, nodeID)
}

// nodeFromContext extracts the node ID set by requireToken.
func nodeFromContext(r *http.Request) string {
	if v, ok := r.Context().Value(nodeIDKey).(string); ok {
		return v
	}
	return ""
}

// logging is a minimal structured request logger.
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
