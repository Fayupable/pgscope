package mysql

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/fayupable/pgscope/internal/domain"
)

// databaseStatsQuery reads four cumulative counters from
// performance_schema.global_status in a single round trip. Com_commit /
// Com_rollback are MySQL's equivalent of Postgres's xact_commit /
// xact_rollback; Innodb_buffer_pool_read_requests / Innodb_buffer_pool_reads
// are the InnoDB buffer pool's logical vs. physical reads, MySQL's
// equivalent of Postgres's blks_hit / blks_read cache hit ratio. MySQL has
// no single counter for temp *bytes* the way Postgres's temp_bytes is —
// Created_tmp_disk_tables is a count, not a byte size, so temp bytes/sec
// is left at zero here rather than reporting a misleading unit.
const databaseStatsQuery = `
SELECT VARIABLE_NAME, VARIABLE_VALUE
FROM performance_schema.global_status
WHERE VARIABLE_NAME IN (
    'Com_commit',
    'Com_rollback',
    'Innodb_buffer_pool_read_requests',
    'Innodb_buffer_pool_reads'
)
`

// DatabaseStatsCollector implements the same rate-tracking pattern as its
// Postgres counterpart: the counters above are cumulative since the server
// started, not meaningful to display live on their own, so this adapter
// keeps the previous measurement in memory and reports the rate of change
// between calls.
type DatabaseStatsCollector struct {
	db *sql.DB

	mu           sync.Mutex
	hasBaseline  bool
	lastCommits  int64
	lastRollback int64
	lastMeasured time.Time
}

func NewDatabaseStatsCollector(db *sql.DB) *DatabaseStatsCollector {
	return &DatabaseStatsCollector{db: db}
}

func (c *DatabaseStatsCollector) FetchDatabaseStats(ctx context.Context) (domain.DatabaseActivityStats, error) {
	counters, err := c.queryCounters(ctx)
	if err != nil {
		return domain.DatabaseActivityStats{}, err
	}

	stats := c.computeRate(counters.commits, counters.rollbacks)
	stats.CacheHitRatio = cacheHitRatio(counters.bufferPoolReadRequests, counters.bufferPoolReads)
	return stats, nil
}

type globalStatusCounters struct {
	commits                int64
	rollbacks              int64
	bufferPoolReadRequests int64
	bufferPoolReads        int64
}

func (c *DatabaseStatsCollector) queryCounters(ctx context.Context) (globalStatusCounters, error) {
	rows, err := c.db.QueryContext(ctx, databaseStatsQuery)
	if err != nil {
		return globalStatusCounters{}, err
	}
	defer func() { _ = rows.Close() }()

	var counters globalStatusCounters
	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return globalStatusCounters{}, err
		}
		switch name {
		case "Com_commit":
			counters.commits = value
		case "Com_rollback":
			counters.rollbacks = value
		case "Innodb_buffer_pool_read_requests":
			counters.bufferPoolReadRequests = value
		case "Innodb_buffer_pool_reads":
			counters.bufferPoolReads = value
		}
	}

	return counters, rows.Err()
}

func (c *DatabaseStatsCollector) computeRate(commits, rollbacks int64) domain.DatabaseActivityStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	if !c.hasBaseline {
		c.setBaseline(commits, rollbacks, now)
		return domain.DatabaseActivityStats{MeasuredAt: now}
	}

	elapsed := now.Sub(c.lastMeasured).Seconds()
	stats := domain.DatabaseActivityStats{
		CommitsPerSecond:   rate(commits-c.lastCommits, elapsed),
		RollbacksPerSecond: rate(rollbacks-c.lastRollback, elapsed),
		MeasuredAt:         now,
	}

	c.setBaseline(commits, rollbacks, now)
	return stats
}

func (c *DatabaseStatsCollector) setBaseline(commits, rollbacks int64, at time.Time) {
	c.hasBaseline = true
	c.lastCommits = commits
	c.lastRollback = rollbacks
	c.lastMeasured = at
}

func rate(delta int64, elapsedSeconds float64) float64 {
	if elapsedSeconds <= 0 {
		return 0
	}
	return float64(delta) / elapsedSeconds
}

// cacheHitRatio reports the fraction of InnoDB buffer pool reads served
// from memory rather than disk, as a 0-100 percentage — same shape and
// same "no reads at all is a perfect ratio" convention as the Postgres
// adapter's identical helper.
func cacheHitRatio(readRequests, physicalReads int64) float64 {
	if readRequests == 0 {
		return 100
	}
	hitRatio := 100 * float64(readRequests-physicalReads) / float64(readRequests)
	if hitRatio < 0 {
		return 0
	}
	return hitRatio
}
