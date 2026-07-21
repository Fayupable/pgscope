package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const statsResetQuery = `
SELECT COALESCE(stats_reset, pg_postmaster_start_time())
FROM pg_stat_database
WHERE datname = current_database()
`

// StatsResetReader turns cumulative counters into per-second rates by
// answering one question: how long has it been since this database's
// statistics were last reset.
type StatsResetReader struct {
	pool *pgxpool.Pool
}

func NewStatsResetReader(pool *pgxpool.Pool) *StatsResetReader {
	return &StatsResetReader{pool: pool}
}

func (r *StatsResetReader) SecondsSinceReset(ctx context.Context) (float64, error) {
	var resetAt time.Time
	if err := r.pool.QueryRow(ctx, statsResetQuery).Scan(&resetAt); err != nil {
		return 0, err
	}
	return time.Since(resetAt).Seconds(), nil
}

func writesPerSecond(writeOps int64, elapsedSeconds float64) float64 {
	if elapsedSeconds <= 0 {
		return 0
	}
	return float64(writeOps) / elapsedSeconds
}
