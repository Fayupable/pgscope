package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const databaseStatsQuery = `
SELECT
    COALESCE(SUM(xact_commit), 0),
    COALESCE(SUM(xact_rollback), 0),
    COALESCE(SUM(blks_hit), 0),
    COALESCE(SUM(blks_read), 0),
    COALESCE(SUM(temp_bytes), 0)
FROM pg_stat_database
WHERE datname = current_database()
`

// DatabaseStatsCollector implements output.IDatabaseStatsCollectorPort
// against Postgres's pg_stat_database view. xact_commit/xact_rollback/
// temp_bytes are cumulative counters since the database was created, so
// this adapter keeps the previous measurement in memory and reports the
// rate of change between calls rather than the raw counters, which are not
// meaningful to display live on their own. blks_hit/blks_read are reported
// as a cumulative ratio instead — cache hit ratio is conventionally read as
// a running average, not a per-tick rate.
type DatabaseStatsCollector struct {
	pool *pgxpool.Pool

	mu           sync.Mutex
	hasBaseline  bool
	lastCommits  int64
	lastRollback int64
	lastTempByte int64
	lastMeasured time.Time
}

func NewDatabaseStatsCollector(pool *pgxpool.Pool) *DatabaseStatsCollector {
	return &DatabaseStatsCollector{pool: pool}
}

func (c *DatabaseStatsCollector) FetchDatabaseStats(ctx context.Context) (domain.DatabaseActivityStats, error) {
	commits, rollbacks, blksHit, blksRead, tempBytes, err := c.queryCounters(ctx)
	if err != nil {
		return domain.DatabaseActivityStats{}, err
	}

	stats := c.computeRate(commits, rollbacks, tempBytes)
	stats.CacheHitRatio = cacheHitRatio(blksHit, blksRead)
	return stats, nil
}

func (c *DatabaseStatsCollector) queryCounters(ctx context.Context) (commits, rollbacks, blksHit, blksRead, tempBytes int64, err error) {
	err = c.pool.QueryRow(ctx, databaseStatsQuery).Scan(&commits, &rollbacks, &blksHit, &blksRead, &tempBytes)
	return commits, rollbacks, blksHit, blksRead, tempBytes, err
}

func (c *DatabaseStatsCollector) computeRate(commits, rollbacks, tempBytes int64) domain.DatabaseActivityStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	if !c.hasBaseline {
		c.setBaseline(commits, rollbacks, tempBytes, now)
		return domain.DatabaseActivityStats{MeasuredAt: now}
	}

	elapsed := now.Sub(c.lastMeasured).Seconds()
	stats := domain.DatabaseActivityStats{
		CommitsPerSecond:   rate(commits-c.lastCommits, elapsed),
		RollbacksPerSecond: rate(rollbacks-c.lastRollback, elapsed),
		TempBytesPerSecond: rate(tempBytes-c.lastTempByte, elapsed),
		MeasuredAt:         now,
	}

	c.setBaseline(commits, rollbacks, tempBytes, now)
	return stats
}

func (c *DatabaseStatsCollector) setBaseline(commits, rollbacks, tempBytes int64, at time.Time) {
	c.hasBaseline = true
	c.lastCommits = commits
	c.lastRollback = rollbacks
	c.lastTempByte = tempBytes
	c.lastMeasured = at
}

func rate(delta int64, elapsedSeconds float64) float64 {
	if elapsedSeconds <= 0 {
		return 0
	}
	return float64(delta) / elapsedSeconds
}

// cacheHitRatio reports the fraction of block reads served from shared
// buffers rather than disk, as a 0-100 percentage. No reads at all is
// treated as a perfect ratio (100), since there was nothing to miss.
func cacheHitRatio(blksHit, blksRead int64) float64 {
	total := blksHit + blksRead
	if total == 0 {
		return 100
	}
	return 100 * float64(blksHit) / float64(total)
}
