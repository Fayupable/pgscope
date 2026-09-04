package mysql

import (
	"context"
	"database/sql"

	"github.com/fayupable/pgscope/internal/domain"
)

const topQueriesLimit = 100

// topQueriesQuery reads the most expensive normalized query shapes from
// performance_schema.events_statements_summary_by_digest — MySQL's
// equivalent of pg_stat_statements. DIGEST_TEXT is already normalized
// (literal values replaced with ?), same privacy property as Postgres's
// pg_stat_statements query column. SUM_TIMER_WAIT/AVG_TIMER_WAIT are in
// picoseconds, converted to milliseconds (÷ 1e9) to match
// domain.SlowQuery's unit, which the Postgres adapter already reports in
// milliseconds.
const topQueriesQuery = `
SELECT
    DIGEST_TEXT,
    COUNT_STAR,
    SUM_TIMER_WAIT / 1000000000,
    AVG_TIMER_WAIT / 1000000000
FROM performance_schema.events_statements_summary_by_digest
WHERE SCHEMA_NAME = DATABASE()
  AND DIGEST_TEXT IS NOT NULL
ORDER BY SUM_TIMER_WAIT DESC
LIMIT ?
`

// TopQueryCollector reads the most expensive normalized query shapes from
// performance_schema. Its only job is fetching that one thing — advisory
// judgments live in their own dedicated collectors, same split as the
// Postgres adapter.
type TopQueryCollector struct {
	db *sql.DB
}

func NewTopQueryCollector(db *sql.DB) *TopQueryCollector {
	return &TopQueryCollector{db: db}
}

func (c *TopQueryCollector) Fetch(ctx context.Context) ([]domain.SlowQuery, error) {
	rows, err := c.db.QueryContext(ctx, topQueriesQuery, topQueriesLimit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	queries := make([]domain.SlowQuery, 0)
	for rows.Next() {
		var q domain.SlowQuery
		if err := rows.Scan(&q.Query, &q.Calls, &q.TotalExecMs, &q.MeanExecMs); err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}

	return queries, rows.Err()
}
