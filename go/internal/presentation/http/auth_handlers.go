package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const sessionMaxAge = 24 * time.Hour

type loginRequest struct {
	Key string `json:"key"`
}

func handleLogin(apiKey string, bans *banStore, attempts *attemptTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		req, err := decodeLoginRequest(r)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}

		if !validKey(req.Key, apiKey) {
			rejectLogin(w, bans, attempts, ip)
			return
		}

		attempts.reset(ip)
		slog.Info("login succeeded", "ip", ip)

		http.SetCookie(w, newSessionCookie(apiKey, int(sessionMaxAge.Seconds())))
		writeJSON(w, map[string]any{"authenticated": true})
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	slog.Info("logout", "ip", clientIP(r))

	http.SetCookie(w, newSessionCookie("", -1))
	writeJSON(w, map[string]any{"authenticated": false})
}

func decodeLoginRequest(r *http.Request) (loginRequest, error) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return loginRequest{}, err
	}
	return req, nil
}

// rejectLogin records the failed attempt, bans the IP once it crosses
// failureBanThreshold, and always responds 401 regardless of ban outcome.
func rejectLogin(w http.ResponseWriter, bans *banStore, attempts *attemptTracker, ip string) {
	count := attempts.recordFailure(ip)
	slog.Warn("login failed: invalid key", "ip", ip, "count", count)

	if count >= failureBanThreshold {
		slog.Warn("banning IP after repeated failed logins", "ip", ip)
		bans.ban(ip, loginBanDuration)
	}

	http.Error(w, "invalid key", http.StatusUnauthorized)
}

// newSessionCookie builds the pgscope_session cookie. handleLogin calls it
// with the real key and a positive MaxAge to set the session; handleLogout
// calls it with an empty value and MaxAge=-1 to clear it. Centralizing this
// means the HttpOnly/Secure/SameSite flags can never drift between the two
// call sites.
func newSessionCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	}
}

func handleAuthStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"authenticated": true})
}
