package http

import (
	"net/http"

	"github.com/fayupable/pgscope/internal/infrastructure/config"
)

// handleConnection reports which database engine this server is currently
// configured against. "id" is always "default" for now — there's only ever
// one connection — but it's included from the start so the frontend can
// key its engine-awareness off a connection id rather than assuming a
// single global engine, and this same shape will describe one entry in a
// future GET /api/v1/connections list once multi-connection support lands,
// with no breaking change to this response.
func handleConnection(engine config.DBEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"id": "default", "engine": engine})
	}
}
