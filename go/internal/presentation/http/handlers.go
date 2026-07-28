package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fayupable/pgscope/internal/application/service"
)

// allowedHistoryWindows are the only replay windows the history endpoint
// accepts — a fixed set, not an arbitrary duration, so a client can't
// request something unreasonably wide.
var allowedHistoryWindows = map[string]time.Duration{
	"1h":  1 * time.Hour,
	"3h":  3 * time.Hour,
	"6h":  6 * time.Hour,
	"12h": 12 * time.Hour,
	"24h": 24 * time.Hour,
}

const defaultHistoryWindow = "1h"

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func handleMonitorStart(poller *service.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		minutes, err := parseMinutesParam(r)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		if err := poller.StartMonitoring(minutes); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{"monitoring": true, "minutes": minutes})
	}
}

func handleMonitorStop(poller *service.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		poller.StopMonitoring()
		writeJSON(w, map[string]any{"monitoring": false})
	}
}

// handleHistory serves the replay window: ?window=1h|3h|6h|12h|24h (defaults
// to 1h). Recording itself isn't a separate action — it happens
// automatically in the background whenever monitoring is active — so this
// endpoint only ever reads what's already been recorded.
func handleHistory(poller *service.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		window, err := parseWindowParam(r)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}

		since := time.Now().Add(-window)
		snapshots, err := poller.RecentHistory(r.Context(), since)
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, snapshots)
	}
}

func parseWindowParam(r *http.Request) (time.Duration, error) {
	raw := r.URL.Query().Get("window")
	if raw == "" {
		raw = defaultHistoryWindow
	}

	window, ok := allowedHistoryWindows[raw]
	if !ok {
		return 0, fmt.Errorf("window must be one of: 1h, 3h, 6h, 12h, 24h, got %q", raw)
	}
	return window, nil
}

func parseMinutesParam(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("minutes")
	if raw == "" {
		return 0, nil
	}
	minutes, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("minutes must be a valid integer, got %q", raw)
	}
	return minutes, nil
}

func handleInsights(insightsService *service.InsightsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		insights, err := insightsService.GetInsights(r.Context())
		if err != nil {
			writeError(w, err, statusForInsightsError(err))
			return
		}
		writeJSON(w, insights)
	}
}

// statusForInsightsError maps a request-timeout (the pool couldn't free a
// connection in time) to 503 — "try again shortly" — rather than the
// generic 500 used for genuine internal errors.
func statusForInsightsError(err error) int {
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}
