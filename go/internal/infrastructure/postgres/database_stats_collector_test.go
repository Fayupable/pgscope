package postgres

import (
	"testing"
	"time"
)

func TestDatabaseStatsCollector_ComputeRate(t *testing.T) {
	c := &DatabaseStatsCollector{}

	t.Run("first call establishes a baseline and reports zero rates", func(t *testing.T) {
		stats := c.computeRate(100, 10, 5000)
		if stats.CommitsPerSecond != 0 || stats.RollbacksPerSecond != 0 || stats.TempBytesPerSecond != 0 {
			t.Errorf("first call = %+v, want all rates zero (no prior baseline to compare against)", stats)
		}
	})

	t.Run("second call computes a rate from the delta since the baseline", func(t *testing.T) {
		c.lastMeasured = time.Now().Add(-2 * time.Second) // force a known elapsed time

		stats := c.computeRate(120, 12, 6000) // +20 commits, +2 rollbacks, +1000 temp bytes over ~2s

		if stats.CommitsPerSecond < 9 || stats.CommitsPerSecond > 11 {
			t.Errorf("CommitsPerSecond = %v, want approx 10", stats.CommitsPerSecond)
		}
		if stats.RollbacksPerSecond < 0.9 || stats.RollbacksPerSecond > 1.1 {
			t.Errorf("RollbacksPerSecond = %v, want approx 1", stats.RollbacksPerSecond)
		}
		if stats.TempBytesPerSecond < 450 || stats.TempBytesPerSecond > 550 {
			t.Errorf("TempBytesPerSecond = %v, want approx 500", stats.TempBytesPerSecond)
		}
	})

	t.Run("counters that haven't moved report a zero rate, not negative or NaN", func(t *testing.T) {
		c.lastMeasured = time.Now().Add(-1 * time.Second)
		stats := c.computeRate(120, 12, 6000) // identical to the previous call's counters

		if stats.CommitsPerSecond != 0 || stats.RollbacksPerSecond != 0 || stats.TempBytesPerSecond != 0 {
			t.Errorf("unchanged counters = %+v, want all rates zero", stats)
		}
	})
}

func TestRate(t *testing.T) {
	tests := []struct {
		name           string
		delta          int64
		elapsedSeconds float64
		want           float64
	}{
		{name: "positive delta over one second", delta: 100, elapsedSeconds: 1, want: 100},
		{name: "positive delta over two seconds", delta: 100, elapsedSeconds: 2, want: 50},
		{name: "zero delta", delta: 0, elapsedSeconds: 5, want: 0},
		{name: "zero elapsed time avoids division by zero", delta: 100, elapsedSeconds: 0, want: 0},
		{name: "negative elapsed time (clock skew) avoids a nonsensical negative rate", delta: 100, elapsedSeconds: -1, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rate(tt.delta, tt.elapsedSeconds); got != tt.want {
				t.Errorf("rate(%d, %v) = %v, want %v", tt.delta, tt.elapsedSeconds, got, tt.want)
			}
		})
	}
}

func TestCacheHitRatio(t *testing.T) {
	tests := []struct {
		name     string
		blksHit  int64
		blksRead int64
		want     float64
	}{
		{name: "no reads at all is treated as a perfect ratio", blksHit: 0, blksRead: 0, want: 100},
		{name: "all hits, no misses", blksHit: 1000, blksRead: 0, want: 100},
		{name: "all misses, no hits", blksHit: 0, blksRead: 1000, want: 0},
		{name: "even split", blksHit: 500, blksRead: 500, want: 50},
		{name: "mostly hits", blksHit: 998, blksRead: 2, want: 99.8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cacheHitRatio(tt.blksHit, tt.blksRead)
			if got < tt.want-0.01 || got > tt.want+0.01 {
				t.Errorf("cacheHitRatio(%d, %d) = %v, want %v", tt.blksHit, tt.blksRead, got, tt.want)
			}
		})
	}
}
