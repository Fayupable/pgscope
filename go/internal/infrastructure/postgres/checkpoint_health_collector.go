package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const checkpointStatsQuery = `
SELECT checkpoints_timed, checkpoints_req
FROM pg_stat_bgwriter
`

// CheckpointHealthCollector reads checkpoint counters directly from
// pg_stat_bgwriter. It has no opinion on what ratio of forced
// checkpoints is "too high" — that judgment belongs to
// domain.NewCheckpointHealth.
type CheckpointHealthCollector struct {
	pool *pgxpool.Pool
}

func NewCheckpointHealthCollector(pool *pgxpool.Pool) *CheckpointHealthCollector {
	return &CheckpointHealthCollector{pool: pool}
}

func (c *CheckpointHealthCollector) Fetch(ctx context.Context) (domain.CheckpointStats, error) {
	var stats domain.CheckpointStats
	err := c.pool.QueryRow(ctx, checkpointStatsQuery).Scan(&stats.ScheduledCheckpoints, &stats.RequestedCheckpoints)
	return stats, err
}
