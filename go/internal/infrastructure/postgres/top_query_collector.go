package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const topQueriesLimit = 100

const topQueriesQuery = `
SELECT query, calls, total_exec_time, mean_exec_time
FROM pg_stat_statements
WHERE ` + currentDbFilter + `
ORDER BY total_exec_time DESC
LIMIT $1
`

// TopQueryCollector reads the most expensive normalized query shapes
// from pg_stat_statements. Its only job is fetching that one thing —
// advisory judgments (index candidates, pagination, etc.) live in their
// own dedicated collectors so they're never limited by this list's size.
type TopQueryCollector struct {
	pool *pgxpool.Pool
}

func NewTopQueryCollector(pool *pgxpool.Pool) *TopQueryCollector {
	return &TopQueryCollector{pool: pool}
}

func (c *TopQueryCollector) Fetch(ctx context.Context) ([]domain.SlowQuery, error) {
	rows, err := c.pool.Query(ctx, topQueriesQuery, topQueriesLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
