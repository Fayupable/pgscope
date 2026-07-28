package history

import (
	"context"
	"sync"
	"time"

	"github.com/fayupable/pgscope/internal/domain"
)

const (
	maxPeriodicSnapshots = 300 // ~5 minutes at a 1-second poll interval
	maxIncidentSnapshots = 50
)

// RingBufferStore implements output.IHistoryStorePort in memory. Periodic
// snapshots are kept in a fixed-size rolling window (oldest evicted first);
// incident snapshots are kept separately and evicted less aggressively since
// they are rarer and more valuable to retain.
type RingBufferStore struct {
	mu        sync.RWMutex
	periodic  []domain.Snapshot
	incidents []domain.Snapshot
}

func NewRingBufferStore() *RingBufferStore {
	return &RingBufferStore{
		periodic:  make([]domain.Snapshot, 0, maxPeriodicSnapshots),
		incidents: make([]domain.Snapshot, 0, maxIncidentSnapshots),
	}
}

func (s *RingBufferStore) Append(_ context.Context, snapshot domain.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.periodic = appendBounded(s.periodic, snapshot, maxPeriodicSnapshots)

	if snapshot.Trigger == domain.SnapshotTriggerIncident {
		s.incidents = appendBounded(s.incidents, snapshot, maxIncidentSnapshots)
	}

	return nil
}

// Recent ignores downsampling entirely — the ring buffer's fixed ~5-minute
// window is already small enough that no window ever needs thinning. since
// simply filters out anything captured before it.
func (s *RingBufferStore) Recent(_ context.Context, since time.Time) ([]domain.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]domain.Snapshot, 0, len(s.periodic))
	for _, snap := range s.periodic {
		if !snap.CapturedAt.Before(since) {
			result = append(result, snap)
		}
	}
	return result, nil
}

func (s *RingBufferStore) Incidents(_ context.Context) ([]domain.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return copySnapshots(s.incidents), nil
}

// appendBounded adds an entry to the end of the slice, evicting the oldest
// entry first if the slice is already at capacity.
func appendBounded(buffer []domain.Snapshot, entry domain.Snapshot, capacity int) []domain.Snapshot {
	if len(buffer) >= capacity {
		buffer = buffer[1:]
	}
	return append(buffer, entry)
}

// copySnapshots returns a defensive copy so callers can't mutate internal
// state through the returned slice.
func copySnapshots(source []domain.Snapshot) []domain.Snapshot {
	result := make([]domain.Snapshot, len(source))
	copy(result, source)
	return result
}
