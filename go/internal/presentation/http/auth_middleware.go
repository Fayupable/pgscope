package http

import (
	"crypto/subtle"
	"net/http"
)

const sessionCookieName = "pgscope_session"

// withAuth wraps a handler so it only runs if the request carries a valid
// session cookie. The cookie's value is the raw API key itself (set by
// handleLogin) — compared in constant time to avoid leaking key length or
// content through response-time differences.
func withAuth(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || !validKey(cookie.Value, apiKey) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validKey(provided, expected string) bool {
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
