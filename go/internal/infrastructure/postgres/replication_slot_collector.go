package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

// replicationSlotQuery reads every replication slot from
// pg_replication_slots, regardless of whether a replica is currently
// connected to consume it — an inactive slot still retains WAL
// indefinitely, which is exactly the failure mode this check exists to
// catch. restart_lsn is NULL for a slot that's never been used yet, so
// COALESCE it against the current WAL position (zero retained bytes,
// nothing to warn about).
const replicationSlotQuery = `
SELECT
    slot_name,
    active,
    pg_wal_lsn_diff(pg_current_wal_lsn(), COALESCE(restart_lsn, pg_current_wal_lsn()))
FROM pg_replication_slots
`

// ReplicationSlotCollector reads replication slot WAL retention directly
// from pg_replication_slots. It has no opinion on what's "too much" —
// that judgment belongs to domain.DetectReplicationSlotWarnings.
type ReplicationSlotCollector struct {
	pool *pgxpool.Pool
}

func NewReplicationSlotCollector(pool *pgxpool.Pool) *ReplicationSlotCollector {
	return &ReplicationSlotCollector{pool: pool}
}

func (c *ReplicationSlotCollector) Fetch(ctx context.Context) ([]domain.ReplicationSlotInfo, error) {
	rows, err := c.pool.Query(ctx, replicationSlotQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make([]domain.ReplicationSlotInfo, 0)
	for rows.Next() {
		var s domain.ReplicationSlotInfo
		if err := rows.Scan(&s.SlotName, &s.Active, &s.RetainedBytes); err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}

	return slots, rows.Err()
}
