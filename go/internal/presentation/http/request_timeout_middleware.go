package http

import (
	"context"
	"net/http"
	"time"
)

// withRequestTimeout bounds the entire request — including any time spent
// waiting for a free connection from the database pool — so a saturated
// pool fails fast instead of leaving the handler's goroutine blocked
// indefinitely.
func withRequestTimeout(d time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
