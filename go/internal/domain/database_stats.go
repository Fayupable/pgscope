package domain

import "time"

// DatabaseActivityStats represents the rate of transaction completion on the
// monitored database, expressed as a per-second rate rather than a raw
// cumulative counter — a raw counter (total since the database started) is
// not meaningful to display live; the rate of change is. CacheHitRatio is
// the exception — it's a cumulative ratio (since last stats reset), which
// is the conventional way this metric is interpreted; a rate-of-change
// version would be noisy and less meaningful than the running average.
type DatabaseActivityStats struct {
	CommitsPerSecond   float64   `json:"commitsPerSecond"`
	RollbacksPerSecond float64   `json:"rollbacksPerSecond"`
	CacheHitRatio      float64   `json:"cacheHitRatio"`
	MeasuredAt         time.Time `json:"measuredAt"`
}
