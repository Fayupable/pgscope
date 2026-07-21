package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// writeError logs the real error server-side but never returns its raw
// text to the client — the underlying error may contain internal details
// (DB driver messages, connection info) that shouldn't be exposed.
func writeError(w http.ResponseWriter, err error, status int) {
	slog.Error("request failed", "error", err, "status", status)
	http.Error(w, http.StatusText(status), status)
}
