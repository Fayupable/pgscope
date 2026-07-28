package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const longRunningQueryQuery = `
SELECT
    a.pid,
    COALESCE(a.usename, ''),
    COALESCE(a.application_name, ''),
    COALESCE(s.query, '[query not tracked by pg_stat_statements]'),
    EXTRACT(EPOCH FROM now() - a.query_start)
FROM pg_stat_activity a
LEFT JOIN pg_stat_statements s ON s.queryid = a.query_id
WHERE a.state = 'active'
  AND a.datname = current_database()
  AND a.pid != pg_backend_pid()
`

// LongRunningQueryCollector reads sessions currently executing a query.
// Query text always comes from pg_stat_statements (parameter values
// replaced with $1, $2, ...), never from pg_stat_activity's raw query
// column — same reasoning as Collector in collector.go: this prevents
// literal values (passwords, PII, tokens) that appear in a running
// statement from ever being exposed through the tool. It has no opinion on
// how long is "too long" — that judgment belongs to
// domain.DetectLongRunningQueryWarnings.
type LongRunningQueryCollector struct {
	pool *pgxpool.Pool
}

func NewLongRunningQueryCollector(pool *pgxpool.Pool) *LongRunningQueryCollector {
	return &LongRunningQueryCollector{pool: pool}
}

func (c *LongRunningQueryCollector) Fetch(ctx context.Context) ([]domain.LongRunningQuerySession, error) {
	rows, err := c.pool.Query(ctx, longRunningQueryQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]domain.LongRunningQuerySession, 0)
	for rows.Next() {
		var s domain.LongRunningQuerySession
		if err := rows.Scan(&s.PID, &s.User, &s.ApplicationName, &s.Query, &s.RunningSeconds); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}
