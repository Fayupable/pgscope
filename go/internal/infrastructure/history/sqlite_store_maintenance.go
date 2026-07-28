package history

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fayupable/pgscope/internal/domain"
)

const (
	pruneBatchSize  = 500
	pruneBatchPause = 50 * time.Millisecond

	// MaxIncidentRows is the recommended cap for PruneExcessIncidents —
	// incidents are never pruned by age, so callers should apply a count
	// cap periodically to bound the history file's growth.
	MaxIncidentRows = 10000
)

// PruneOlderThan deletes periodic snapshots captured before the given time,
// in small batches so a large backlog never locks the database for long or
// causes a CPU/memory spike. Incident snapshots are never pruned this way —
// they're rare and valuable enough to keep regardless of age.
func (s *SQLiteStore) PruneOlderThan(ctx context.Context, before time.Time) error {
	for {
		result, err := s.db.ExecContext(ctx,
			`DELETE FROM snapshots WHERE id IN (
				SELECT id FROM snapshots
				WHERE captured_at < ? AND trigger = ?
				LIMIT ?
			)`,
			before, string(domain.SnapshotTriggerPeriodic), pruneBatchSize,
		)
		if err != nil {
			return fmt.Errorf("prune old snapshots: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read prune batch result: %w", err)
		}
		if rowsAffected == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pruneBatchPause):
		}
	}
}

// PruneExcessIncidents keeps only the most recent maxRows incident
// snapshots, deleting the rest in the same small-batch style as
// PruneOlderThan. Unlike periodic snapshots, incidents are never pruned by
// age — this is a count-based safety net so a database with a persistently
// noisy blocking pattern can't grow the history file without bound.
func (s *SQLiteStore) PruneExcessIncidents(ctx context.Context, maxRows int) error {
	var count int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM snapshots WHERE trigger = ?`,
		string(domain.SnapshotTriggerIncident),
	).Scan(&count); err != nil {
		return fmt.Errorf("count incident snapshots: %w", err)
	}
	if count <= maxRows {
		return nil
	}

	var cutoff time.Time
	if err := s.db.QueryRowContext(ctx,
		`SELECT captured_at FROM snapshots
			WHERE trigger = ?
			ORDER BY captured_at DESC
			LIMIT 1 OFFSET ?`,
		string(domain.SnapshotTriggerIncident), maxRows-1,
	).Scan(&cutoff); err != nil {
		return fmt.Errorf("find incident prune cutoff: %w", err)
	}

	for {
		result, err := s.db.ExecContext(ctx,
			`DELETE FROM snapshots WHERE id IN (
				SELECT id FROM snapshots
				WHERE captured_at < ? AND trigger = ?
				LIMIT ?
			)`,
			cutoff, string(domain.SnapshotTriggerIncident), pruneBatchSize,
		)
		if err != nil {
			return fmt.Errorf("prune excess incident snapshots: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read incident prune batch result: %w", err)
		}
		if rowsAffected == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pruneBatchPause):
		}
	}
}

// EnforceMaxSize is a backstop on top of age-based retention
// (PruneOlderThan) — if the history file's on-disk size exceeds maxBytes
// (an unusually busy monitored database, a misconfigured record interval,
// ...), the oldest periodic snapshots are deleted first. If that alone
// isn't enough to get back under the cap (periodic snapshots exhausted but
// the file is still too large — only realistic during a sustained, severe
// blocking storm that outpaces PruneExcessIncidents' count cap), the
// oldest incidents are pruned too. The disk-size guarantee holds
// regardless of trigger type; PruneExcessIncidents remains the first line
// of defense for incident growth under normal conditions.
func (s *SQLiteStore) EnforceMaxSize(ctx context.Context, maxBytes int64) error {
	size, err := s.fileSize()
	if err != nil {
		return fmt.Errorf("stat history db file: %w", err)
	}

	size, err = s.pruneUntilUnderSize(ctx, maxBytes, size, domain.SnapshotTriggerPeriodic)
	if err != nil {
		return err
	}

	if size > maxBytes {
		if _, err := s.pruneUntilUnderSize(ctx, maxBytes, size, domain.SnapshotTriggerIncident); err != nil {
			return err
		}
	}

	return nil
}

// pruneUntilUnderSize deletes the oldest snapshots matching trigger, in
// small batches (with an incremental_vacuum after each, since SQLite
// otherwise never actually shrinks the file on disk), until the file is
// back under maxBytes or there's nothing left of that trigger type to
// delete — whichever comes first. Returns the file's resulting size either
// way, so the caller can decide whether a second pass (a different
// trigger) is needed.
func (s *SQLiteStore) pruneUntilUnderSize(ctx context.Context, maxBytes, currentSize int64, trigger domain.SnapshotTrigger) (int64, error) {
	size := currentSize

	for size > maxBytes {
		result, err := s.db.ExecContext(ctx,
			`DELETE FROM snapshots WHERE id IN (
				SELECT id FROM snapshots
				WHERE trigger = ?
				ORDER BY captured_at ASC
				LIMIT ?
			)`,
			string(trigger), pruneBatchSize,
		)
		if err != nil {
			return size, fmt.Errorf("prune %s snapshots to enforce max size: %w", trigger, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return size, fmt.Errorf("read size-enforcement prune batch result: %w", err)
		}
		if rowsAffected == 0 {
			return size, nil
		}

		if _, err := s.db.ExecContext(ctx, `PRAGMA incremental_vacuum`); err != nil {
			return size, fmt.Errorf("incremental vacuum: %w", err)
		}

		select {
		case <-ctx.Done():
			return size, ctx.Err()
		case <-time.After(pruneBatchPause):
		}

		size, err = s.fileSize()
		if err != nil {
			return size, fmt.Errorf("stat history db file: %w", err)
		}
	}

	return size, nil
}

// fileSize reports the history file's true on-disk size. In WAL mode,
// recent writes (and the space freed by recent deletes/incremental_vacuum)
// can sit in the separate -wal sidecar file rather than the main file for a
// while — os.Stat on the main path alone would badly undercount actual
// usage until SQLite's own automatic checkpoint eventually catches up. A
// checkpoint is forced first so the main file's size always reflects
// reality at the moment this is called.
func (s *SQLiteStore) fileSize() (int64, error) {
	if _, err := s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return 0, fmt.Errorf("checkpoint before measuring size: %w", err)
	}

	info, err := os.Stat(s.path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
