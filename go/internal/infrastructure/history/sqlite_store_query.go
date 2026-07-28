package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fayupable/pgscope/internal/domain"
)

// maxPeriodicRowsPerWindow bounds how many periodic snapshots Recent ever
// returns for a single window, regardless of how wide the window is or how
// dense the underlying data is. This is sized for what a replay
// chart/scrubber can usefully render — far more points than this just
// overlap on screen — and keeps the response payload bounded. Windows with
// more raw data than this are evenly downsampled rather than truncated, so
// the whole window stays represented instead of only its tail.
const maxPeriodicRowsPerWindow = 1500

// Recent returns every incident snapshot in the window untouched, plus
// periodic snapshots downsampled (if needed) so their count never exceeds
// maxPeriodicRowsPerWindow — see that constant's comment for why. The two
// sets are merged and returned in chronological order.
func (s *SQLiteStore) Recent(ctx context.Context, since time.Time) ([]domain.Snapshot, error) {
	periodic, err := s.recentPeriodic(ctx, since)
	if err != nil {
		return nil, err
	}

	incidents, err := s.recentIncidents(ctx, since)
	if err != nil {
		return nil, err
	}

	merged := mergeSnapshotsByTime(periodic, incidents)
	return merged, nil
}

func (s *SQLiteStore) recentPeriodic(ctx context.Context, since time.Time) ([]domain.Snapshot, error) {
	var count int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM snapshots WHERE trigger = ? AND captured_at >= ?`,
		string(domain.SnapshotTriggerPeriodic), since,
	).Scan(&count); err != nil {
		return nil, fmt.Errorf("count periodic snapshots in window: %w", err)
	}

	if count == 0 {
		return []domain.Snapshot{}, nil
	}

	step := 1
	if count > maxPeriodicRowsPerWindow {
		step = (count + maxPeriodicRowsPerWindow - 1) / maxPeriodicRowsPerWindow
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT payload FROM (
			SELECT payload, ROW_NUMBER() OVER (ORDER BY captured_at) AS rn
			FROM snapshots
			WHERE trigger = ? AND captured_at >= ?
		)
		WHERE (rn - 1) % ? = 0
		ORDER BY rn
	`, string(domain.SnapshotTriggerPeriodic), since, step)
	if err != nil {
		return nil, fmt.Errorf("query periodic snapshots in window: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanSnapshots(rows)
}

func (s *SQLiteStore) recentIncidents(ctx context.Context, since time.Time) ([]domain.Snapshot, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT payload FROM snapshots WHERE trigger = ? AND captured_at >= ? ORDER BY captured_at ASC`,
		string(domain.SnapshotTriggerIncident), since,
	)
	if err != nil {
		return nil, fmt.Errorf("query incident snapshots in window: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanSnapshots(rows)
}

// mergeSnapshotsByTime merges two already-sorted-by-time slices into one
// chronologically ordered slice (a standard merge step, not a full sort,
// since both inputs are already ordered).
func mergeSnapshotsByTime(a, b []domain.Snapshot) []domain.Snapshot {
	merged := make([]domain.Snapshot, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i].CapturedAt.Before(b[j].CapturedAt) {
			merged = append(merged, a[i])
			i++
		} else {
			merged = append(merged, b[j])
			j++
		}
	}
	merged = append(merged, a[i:]...)
	merged = append(merged, b[j:]...)
	return merged
}

func (s *SQLiteStore) Incidents(ctx context.Context) ([]domain.Snapshot, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT payload FROM snapshots WHERE trigger = ? ORDER BY captured_at ASC`,
		string(domain.SnapshotTriggerIncident),
	)
	if err != nil {
		return nil, fmt.Errorf("query incident snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanSnapshots(rows)
}

func scanSnapshots(rows *sql.Rows) ([]domain.Snapshot, error) {
	snapshots := make([]domain.Snapshot, 0)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan snapshot row: %w", err)
		}

		var snapshot domain.Snapshot
		if err := json.Unmarshal([]byte(payload), &snapshot); err != nil {
			return nil, fmt.Errorf("unmarshal snapshot payload: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot rows: %w", err)
	}
	return snapshots, nil
}
