package http

import "sync"

// attemptTracker counts consecutive failures per IP. A single mistake
// (mistyped key, one stray bot request) never triggers a ban on its own —
// only once the count crosses a threshold does the caller decide to ban.
type attemptTracker struct {
	mu     sync.Mutex
	counts map[string]int
}

func newAttemptTracker() *attemptTracker {
	return &attemptTracker{counts: make(map[string]int)}
}

// recordFailure increments the count for ip and returns the new total.
func (t *attemptTracker) recordFailure(ip string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts[ip]++
	return t.counts[ip]
}

// reset clears the count for ip — called on a successful login so a past
// mistake doesn't count against a client indefinitely.
func (t *attemptTracker) reset(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.counts, ip)
}
