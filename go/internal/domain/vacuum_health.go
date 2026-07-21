package domain

import (
	"fmt"
	"time"
)

const (
	DeadTupleRatioWarningPercent = 20.0
	MinLiveTuplesForVacuumCheck  = 1000
)

// TableVacuumStats is the raw shape read from pg_stat_user_tables — no
// judgment applied yet.
type TableVacuumStats struct {
	Table          string
	LiveTuples     int64
	DeadTuples     int64
	LastAutovacuum *time.Time
}

func (s TableVacuumStats) deadTupleRatio() float64 {
	total := s.LiveTuples + s.DeadTuples
	if total == 0 {
		return 0
	}
	return 100 * float64(s.DeadTuples) / float64(total)
}

// VacuumHealthWarning is a suggestion, never a certainty — a high dead
// tuple ratio slows down sequential scans and wastes disk space, but a
// table with heavy legitimate churn will always have some dead tuples.
type VacuumHealthWarning struct {
	Table          string     `json:"table"`
	DeadTupleRatio float64    `json:"deadTupleRatio"`
	LastAutovacuum *time.Time `json:"lastAutovacuum,omitempty"`
	Explanation    string     `json:"explanation"`
}

// DetectVacuumHealthWarnings filters tables down to the ones with a
// meaningfully high dead-tuple ratio, ignoring small tables where the
// ratio is noisy and the absolute waste is negligible.
func DetectVacuumHealthWarnings(stats []TableVacuumStats) []VacuumHealthWarning {
	result := make([]VacuumHealthWarning, 0)
	for _, s := range stats {
		if s.LiveTuples < MinLiveTuplesForVacuumCheck {
			continue
		}

		ratio := s.deadTupleRatio()
		if ratio < DeadTupleRatioWarningPercent {
			continue
		}

		result = append(result, VacuumHealthWarning{
			Table:          s.Table,
			DeadTupleRatio: ratio,
			LastAutovacuum: s.LastAutovacuum,
			Explanation:    buildVacuumExplanation(s, ratio),
		})
	}
	return result
}

func buildVacuumExplanation(s TableVacuumStats, ratio float64) string {
	total := s.LiveTuples + s.DeadTuples
	if s.LastAutovacuum == nil {
		return fmt.Sprintf(
			"Table %q has %.0f%% dead tuples (%d of %d rows) and has never been autovacuumed. This can slow down queries and bloat the table on disk — check autovacuum settings for this table.",
			s.Table, ratio, s.DeadTuples, total,
		)
	}
	return fmt.Sprintf(
		"Table %q has %.0f%% dead tuples (%d of %d rows). Last autovacuumed at %s. If this keeps climbing, consider tuning autovacuum thresholds for this table.",
		s.Table, ratio, s.DeadTuples, total, s.LastAutovacuum.Format(time.RFC3339),
	)
}
