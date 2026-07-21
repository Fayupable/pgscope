package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fayupable/pgscope/internal/application/service"
)

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

func handleRecordStart(poller *service.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		minutes, err := parseMinutesParam(r)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		if err := poller.StartRecording(minutes); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{"recording": true, "minutes": minutes})
	}
}

func handleRecordStop(poller *service.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		poller.StopRecording()
		writeJSON(w, map[string]any{"recording": false})
	}
}

func handleHistory(poller *service.Poller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshots, err := poller.RecentHistory(r.Context())
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, snapshots)
	}
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
