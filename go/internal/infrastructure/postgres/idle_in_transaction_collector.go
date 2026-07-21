package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const idleInTransactionQuery = `
SELECT
    pid,
    COALESCE(usename, ''),
    COALESCE(application_name, ''),
    EXTRACT(EPOCH FROM now() - state_change)
FROM pg_stat_activity
WHERE state = 'idle in transaction'
  AND datname = current_database()
  AND pid != pg_backend_pid()
`

// IdleInTransactionCollector reads sessions currently sitting idle
// inside an open transaction, straight from pg_stat_activity. It has no
// opinion on how long is "too long" — that judgment belongs to
// domain.DetectIdleInTransactionWarnings.
type IdleInTransactionCollector struct {
	pool *pgxpool.Pool
}

func NewIdleInTransactionCollector(pool *pgxpool.Pool) *IdleInTransactionCollector {
	return &IdleInTransactionCollector{pool: pool}
}

func (c *IdleInTransactionCollector) Fetch(ctx context.Context) ([]domain.IdleInTransactionSession, error) {
	rows, err := c.pool.Query(ctx, idleInTransactionQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]domain.IdleInTransactionSession, 0)
	for rows.Next() {
		var s domain.IdleInTransactionSession
		if err := rows.Scan(&s.PID, &s.User, &s.ApplicationName, &s.IdleSeconds); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}
