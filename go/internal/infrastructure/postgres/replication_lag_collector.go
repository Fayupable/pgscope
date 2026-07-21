package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

// replicationLagQuery reads connected replicas from pg_stat_replication.
// This returns zero rows on a primary with no replicas — which is the
// current dev setup, so this feature can't be exercised locally yet.
const replicationLagQuery = `
SELECT
    COALESCE(application_name, ''),
    COALESCE(client_addr::text, ''),
    pg_wal_lsn_diff(sent_lsn, replay_lsn)
FROM pg_stat_replication
`

// ReplicationLagCollector reads replica lag directly from
// pg_stat_replication. It has no opinion on what lag is "too much" —
// that judgment belongs to domain.DetectReplicationLagWarnings.
type ReplicationLagCollector struct {
	pool *pgxpool.Pool
}

func NewReplicationLagCollector(pool *pgxpool.Pool) *ReplicationLagCollector {
	return &ReplicationLagCollector{pool: pool}
}

func (c *ReplicationLagCollector) Fetch(ctx context.Context) ([]domain.ReplicaLagInfo, error) {
	rows, err := c.pool.Query(ctx, replicationLagQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	replicas := make([]domain.ReplicaLagInfo, 0)
	for rows.Next() {
		var r domain.ReplicaLagInfo
		if err := rows.Scan(&r.ApplicationName, &r.ClientAddr, &r.LagBytes); err != nil {
			return nil, err
		}
		replicas = append(replicas, r)
	}

	return replicas, rows.Err()
}
