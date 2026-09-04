package http

import (
	"net/http"
	"time"

	"github.com/fayupable/pgscope/internal/application/service"
	"github.com/fayupable/pgscope/internal/infrastructure/config"
	"github.com/fayupable/pgscope/internal/infrastructure/sse"
)

const insightsTimeout = 5 * time.Second

func NewRouter(broadcaster *sse.Broadcaster, poller *service.Poller, insightsService *service.InsightsService, cfg config.Config) http.Handler {
	mux := http.NewServeMux()

	bans := newBanStore()
	loginAttempts := newAttemptTracker()
	scanAttempts := newAttemptTracker()

	loginLimiter := newRateLimiter(0.1, 3)
	insightsLimiter := newRateLimiter(cfg.InsightsRateLimitPerSecond, cfg.InsightsRateLimitBurst)
	generalLimiter := newRateLimiter(2, 5)

	mux.HandleFunc("GET /healthz", handleHealth)
	mux.Handle("POST /api/v1/auth/login", withRateLimit(loginLimiter, handleLogin(cfg.APIKey, bans, loginAttempts)))
	mux.HandleFunc("POST /api/v1/auth/logout", handleLogout)
	mux.Handle("GET /api/v1/auth/status", withAuth(cfg.APIKey, http.HandlerFunc(handleAuthStatus)))
	mux.Handle("GET /api/v1/connection", withAuth(cfg.APIKey, handleConnection(cfg.Engine)))

	// broadcaster and poller are nil only when a wiring path (see main.go)
	// deliberately doesn't build them — every engine currently in main.go
	// builds both, but this guard is what lets a future engine ship with
	// just insights before its ISessionCollectorPort adapter exists,
	// exactly how MySQL itself shipped in an earlier phase of this project.
	if broadcaster != nil {
		mux.Handle("GET /api/v1/sessions/stream", withAuth(cfg.APIKey, broadcaster))
	}

	if poller != nil {
		mux.Handle("POST /api/v1/monitor/start", withAuth(cfg.APIKey, withRateLimit(generalLimiter, handleMonitorStart(poller))))
		mux.Handle("POST /api/v1/monitor/stop", withAuth(cfg.APIKey, withRateLimit(generalLimiter, handleMonitorStop(poller))))
		mux.Handle("GET /api/v1/history", withAuth(cfg.APIKey, withRateLimit(generalLimiter, handleHistory(poller))))
	}

	if insightsService != nil {
		mux.Handle("GET /api/v1/insights", withAuth(cfg.APIKey, withRateLimit(insightsLimiter, withRequestTimeout(insightsTimeout, handleInsights(insightsService)))))
	}

	mux.HandleFunc("/", handleScanProbe(bans, scanAttempts))

	return withBanCheck(bans, mux)
}
