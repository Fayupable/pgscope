package mysql

import (
	"context"
	"database/sql"

	"github.com/fayupable/pgscope/internal/domain"
)

const paginationCandidateLimit = 50

// paginationCandidatesQuery mirrors the Postgres adapter's filter: any
// normalized query shape containing OFFSET, ranked by call count rather
// than cost, so a cheap or rarely-ranked shape is just as visible as an
// expensive one. AVG_TIMER_WAIT/MAX_TIMER_WAIT are in picoseconds,
// converted to milliseconds (÷ 1e9) to match domain.PaginationSignal's
// unit.
//
// MySQL's events_statements_summary_by_digest has no stddev column —
// unlike pg_stat_statements.stddev_exec_time, which the Postgres adapter
// uses directly. As an approximation, (MAX_TIMER_WAIT - AVG_TIMER_WAIT)
// is passed into domain.PaginationSignal's StddevExecMs field: it isn't a
// true standard deviation, but it captures the same underlying signal —
// a query shape whose slowest run was far worse than its average run,
// consistent with deep-OFFSET pagination getting progressively slower.
const paginationCandidatesQuery = `
SELECT
    DIGEST_TEXT,
    COUNT_STAR,
    AVG_TIMER_WAIT / 1000000000,
    (MAX_TIMER_WAIT - AVG_TIMER_WAIT) / 1000000000,
    SUM_ROWS_SENT
FROM performance_schema.events_statements_summary_by_digest
WHERE SCHEMA_NAME = DATABASE()
  AND DIGEST_TEXT LIKE '%OFFSET%'
ORDER BY COUNT_STAR DESC
LIMIT ?
`

// PaginationCollector reads OFFSET-containing query statistics from
// performance_schema. It has no opinion on whether a pattern is worth
// warning about — that judgment belongs to domain.DetectPaginationWarnings,
// unchanged from the Postgres adapter.
type PaginationCollector struct {
	db *sql.DB
}

func NewPaginationCollector(db *sql.DB) *PaginationCollector {
	return &PaginationCollector{db: db}
}

func (c *PaginationCollector) FetchCandidates(ctx context.Context) ([]domain.PaginationSignal, error) {
	rows, err := c.db.QueryContext(ctx, paginationCandidatesQuery, paginationCandidateLimit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	signals := make([]domain.PaginationSignal, 0)
	for rows.Next() {
		var s domain.PaginationSignal
		if err := rows.Scan(&s.Query, &s.Calls, &s.MeanExecMs, &s.StddevExecMs, &s.Rows); err != nil {
			return nil, err
		}
		s.ContainsOffset = true // guaranteed by the SQL filter above
		signals = append(signals, s)
	}

	return signals, rows.Err()
}
