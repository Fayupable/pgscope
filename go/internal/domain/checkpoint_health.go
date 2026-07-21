package domain

import "fmt"

const ForcedCheckpointRatioWarningPercent = 10.0

// CheckpointStats is the raw shape read from pg_stat_bgwriter — no
// judgment applied yet. Note: this view's checkpoint columns moved to
// pg_stat_checkpointer in PostgreSQL 17+; this project currently
// targets PostgreSQL 16, so pg_stat_bgwriter is the correct source.
type CheckpointStats struct {
	ScheduledCheckpoints int64
	RequestedCheckpoints int64
}

func (s CheckpointStats) total() int64 {
	return s.ScheduledCheckpoints + s.RequestedCheckpoints
}

func (s CheckpointStats) requestedRatio() float64 {
	if s.total() == 0 {
		return 0
	}
	return 100 * float64(s.RequestedCheckpoints) / float64(s.total())
}

// CheckpointHealth is a suggestion, never a certainty — a high share of
// forced (requested) checkpoints usually means max_wal_size is too
// small for the current write volume, causing more frequent and
// disruptive checkpoints than necessary.
type CheckpointHealth struct {
	ScheduledCheckpoints int64   `json:"scheduledCheckpoints"`
	RequestedCheckpoints int64   `json:"requestedCheckpoints"`
	RequestedRatio       float64 `json:"requestedRatio"`
	Warning              string  `json:"warning,omitempty"`
}

// NewCheckpointHealth computes the forced-checkpoint ratio and, if it's
// high enough to matter, a warning message.
func NewCheckpointHealth(stats CheckpointStats) CheckpointHealth {
	ratio := stats.requestedRatio()

	warning := ""
	if stats.total() > 0 && ratio >= ForcedCheckpointRatioWarningPercent {
		warning = fmt.Sprintf(
			"%.0f%% of checkpoints (%d of %d) have been forced rather than on schedule. This usually means max_wal_size is too small for the current write volume, causing more frequent, disruptive checkpoints. Consider increasing max_wal_size.",
			ratio, stats.RequestedCheckpoints, stats.total(),
		)
	}

	return CheckpointHealth{
		ScheduledCheckpoints: stats.ScheduledCheckpoints,
		RequestedCheckpoints: stats.RequestedCheckpoints,
		RequestedRatio:       ratio,
		Warning:              warning,
	}
}
