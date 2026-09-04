package mysql

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

const indexCandidateLimit = 10

// indexCandidateTablesQuery is MySQL's equivalent of the Postgres adapter's
// pg_stat_user_tables read. performance_schema.table_io_waits_summary_by_index_usage
// breaks table access down per index, with one special row per table where
// INDEX_NAME IS NULL — that row represents access that didn't go through
// any index at all, MySQL's equivalent of a Postgres sequential scan.
// Summing COUNT_FETCH across the NULL-index row vs every named-index row
// gives the same seq-scan-vs-idx-scan signal domain.IndexSignal expects.
// TABLE_ROWS from information_schema.TABLES is an estimate, same caveat
// Postgres's n_live_tup carries.
const indexCandidateTablesQuery = `
SELECT
    w.OBJECT_NAME,
    SUM(CASE WHEN w.INDEX_NAME IS NULL THEN w.COUNT_FETCH ELSE 0 END) AS seq_scan,
    SUM(CASE WHEN w.INDEX_NAME IS NOT NULL THEN w.COUNT_FETCH ELSE 0 END) AS idx_scan,
    MAX(t.TABLE_ROWS) AS estimated_rows,
    SUM(w.COUNT_INSERT + w.COUNT_UPDATE) AS write_ops,
    COUNT(DISTINCT w.INDEX_NAME) AS index_count
FROM performance_schema.table_io_waits_summary_by_index_usage w
JOIN information_schema.TABLES t
    ON t.TABLE_SCHEMA = w.OBJECT_SCHEMA AND t.TABLE_NAME = w.OBJECT_NAME
WHERE w.OBJECT_SCHEMA = DATABASE()
GROUP BY w.OBJECT_NAME
HAVING idx_scan > 0
   AND estimated_rows >= ?
   AND (100.0 * idx_scan / NULLIF(seq_scan + idx_scan, 0)) < ?
ORDER BY seq_scan DESC
LIMIT ?
`

// RawIndexCandidate is the unjudged shape read from performance_schema —
// domain.IndexSignal decides what, if anything, these numbers mean.
type RawIndexCandidate struct {
	Table         string
	SeqScan       int64
	IdxScan       int64
	EstimatedRows int64
	WriteOps      int64
	IndexCount    int
}

// IndexCandidateCollector reads raw scan/write statistics per table. Like
// the MySQL DatabaseStatsCollector, it's stateful: WriteOps is a
// cumulative counter since the server started, not meaningful on its
// own, so this adapter tracks the previous measurement per table and
// reports a writes/second rate between calls.
type IndexCandidateCollector struct {
	db *sql.DB

	mu           sync.Mutex
	lastWriteOps map[string]int64
	lastMeasured time.Time
	hasBaseline  bool
}

func NewIndexCandidateCollector(db *sql.DB) *IndexCandidateCollector {
	return &IndexCandidateCollector{
		db:           db,
		lastWriteOps: make(map[string]int64),
	}
}

func (c *IndexCandidateCollector) FetchRaw(ctx context.Context, minRows int64, maxIndexUsagePercent float64) ([]RawIndexCandidate, error) {
	rows, err := c.db.QueryContext(ctx, indexCandidateTablesQuery, minRows, maxIndexUsagePercent, indexCandidateLimit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	raw := make([]RawIndexCandidate, 0)
	for rows.Next() {
		var rc RawIndexCandidate
		if err := rows.Scan(&rc.Table, &rc.SeqScan, &rc.IdxScan, &rc.EstimatedRows, &rc.WriteOps, &rc.IndexCount); err != nil {
			return nil, err
		}
		raw = append(raw, rc)
	}

	return raw, rows.Err()
}

// BeginCycle marks the start of one fetch cycle and returns the elapsed
// time since the previous cycle, to be reused across every table's
// WritesPerSecond call within this cycle. Calling it once per cycle
// (rather than once per table) prevents the second table measured in
// the same cycle from seeing a near-zero elapsed time against the first.
func (c *IndexCandidateCollector) BeginCycle() (elapsedSeconds float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if c.hasBaseline {
		elapsedSeconds = now.Sub(c.lastMeasured).Seconds()
	}
	c.lastMeasured = now
	c.hasBaseline = true
	return elapsedSeconds
}

// WritesPerSecond returns the rate of change in a table's write_ops
// counter since the previous cycle, using the elapsed time BeginCycle
// reported for the current cycle. A table seen for the first time has
// nothing to compare against, so it returns 0 and records a baseline.
func (c *IndexCandidateCollector) WritesPerSecond(table string, writeOps int64, elapsedSeconds float64) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	previous, seen := c.lastWriteOps[table]
	c.lastWriteOps[table] = writeOps

	if !seen || elapsedSeconds <= 0 {
		return 0
	}
	return float64(writeOps-previous) / elapsedSeconds
}
