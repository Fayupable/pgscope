package http

import (
	"sync"
	"time"
)

// banStore tracks IPs temporarily blocked after repeated failed login
// attempts or repeated probes of known vulnerability-scanner paths.
// In-memory only — bans do not survive a process restart. Accepted
// tradeoff for this project's single-instance scale; a restart is
// effectively "unban everyone."
type banStore struct {
	mu          sync.Mutex
	bannedUntil map[string]time.Time
}

func newBanStore() *banStore {
	return &banStore{bannedUntil: make(map[string]time.Time)}
}

func (b *banStore) ban(ip string, duration time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.bannedUntil[ip] = time.Now().Add(duration)
}

func (b *banStore) isBanned(ip string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	until, ok := b.bannedUntil[ip]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(b.bannedUntil, ip)
		return false
	}
	return true
}
