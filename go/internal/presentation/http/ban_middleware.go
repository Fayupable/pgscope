package http

import (
	"log/slog"
	"net/http"
	"time"
)

const (
	loginBanDuration    = 7 * 24 * time.Hour
	scanBanDuration     = 7 * 24 * time.Hour
	failureBanThreshold = 4
)

// withBanCheck rejects every request from a banned IP before it reaches
// auth, rate limiting, or any handler logic — this must wrap the outermost
// layer so a banned client can't even burn a rate-limit token.
func withBanCheck(bans *banStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bans.isBanned(clientIP(r)) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// knownScannerPaths are URLs bots/vulnerability scanners probe on virtually
// every exposed IP on the internet (WordPress, PHP, cloud credential files,
// common admin panels). pgscope serves none of these — repeated requests
// for them are automated abuse, not a legitimate client mistake.
var knownScannerPaths = map[string]bool{
	"/.env":                      true,
	"/.git/config":               true,
	"/wp-login.php":              true,
	"/wp-admin/":                 true,
	"/wp-admin/setup-config.php": true,
	"/xmlrpc.php":                true,
	"/phpmyadmin":                true,
	"/phpMyAdmin":                true,
	"/.aws/credentials":          true,
	"/actuator/health":           true,
	"/config.php":                true,
	"/.ssh/id_rsa":               true,
	"/admin.php":                 true,
	"/.docker/config.json":       true,
}

// handleScanProbe serves as the catch-all for any path not otherwise
// routed. A single hit on a known scanner path is logged but not banned
// outright — only after failureBanThreshold hits (a real scanner sweeping
// multiple known paths, not a one-off) does the IP get banned.
func handleScanProbe(bans *banStore, attempts *attemptTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if knownScannerPaths[r.URL.Path] {
			ip := clientIP(r)
			count := attempts.recordFailure(ip)
			slog.Warn("scanner path probed", "ip", ip, "path", r.URL.Path, "count", count)

			if count >= failureBanThreshold {
				slog.Warn("banning IP after repeated scanner probes", "ip", ip)
				bans.ban(ip, scanBanDuration)
			}
		}
		http.NotFound(w, r)
	}
}
