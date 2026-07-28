package domain

import "testing"

func TestNewCheckpointHealth(t *testing.T) {
	tests := []struct {
		name        string
		stats       CheckpointStats
		wantRatio   float64
		wantWarning bool
	}{
		{
			name:        "no checkpoints at all yields zero ratio and no warning",
			stats:       CheckpointStats{ScheduledCheckpoints: 0, RequestedCheckpoints: 0},
			wantRatio:   0,
			wantWarning: false,
		},
		{
			name:        "all scheduled, none forced, is healthy",
			stats:       CheckpointStats{ScheduledCheckpoints: 100, RequestedCheckpoints: 0},
			wantRatio:   0,
			wantWarning: false,
		},
		{
			name:        "all forced, none scheduled, is one hundred percent",
			stats:       CheckpointStats{ScheduledCheckpoints: 0, RequestedCheckpoints: 100},
			wantRatio:   100,
			wantWarning: true,
		},
		{
			name:        "below the warning threshold has a ratio but no warning",
			stats:       CheckpointStats{ScheduledCheckpoints: 95, RequestedCheckpoints: 5}, // 5%
			wantRatio:   5,
			wantWarning: false,
		},
		{
			name:        "exactly at the warning threshold triggers a warning",
			stats:       CheckpointStats{ScheduledCheckpoints: 90, RequestedCheckpoints: 10}, // exactly 10%
			wantRatio:   10,
			wantWarning: true,
		},
		{
			name:        "above the warning threshold triggers a warning",
			stats:       CheckpointStats{ScheduledCheckpoints: 50, RequestedCheckpoints: 50}, // 50%
			wantRatio:   50,
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewCheckpointHealth(tt.stats)

			if got.RequestedRatio != tt.wantRatio {
				t.Errorf("RequestedRatio = %v, want %v", got.RequestedRatio, tt.wantRatio)
			}
			hasWarning := got.Warning != ""
			if hasWarning != tt.wantWarning {
				t.Errorf("Warning present = %v (%q), want present = %v", hasWarning, got.Warning, tt.wantWarning)
			}
			if got.ScheduledCheckpoints != tt.stats.ScheduledCheckpoints {
				t.Errorf("ScheduledCheckpoints = %d, want %d", got.ScheduledCheckpoints, tt.stats.ScheduledCheckpoints)
			}
			if got.RequestedCheckpoints != tt.stats.RequestedCheckpoints {
				t.Errorf("RequestedCheckpoints = %d, want %d", got.RequestedCheckpoints, tt.stats.RequestedCheckpoints)
			}
		})
	}
}

func TestNewCheckpointHealth_WarningContent(t *testing.T) {
	got := NewCheckpointHealth(CheckpointStats{ScheduledCheckpoints: 20, RequestedCheckpoints: 80})

	for _, want := range []string{"80%", "80 of 100", "max_wal_size"} {
		if !contains(got.Warning, want) {
			t.Errorf("Warning = %q, want it to contain %q", got.Warning, want)
		}
	}
}
