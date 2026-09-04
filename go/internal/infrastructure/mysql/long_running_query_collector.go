package mysql

import (
	"context"
	"database/sql"

	"github.com/fayupable/pgscope/internal/domain"
)

// longRunningQueryQuery reads sessions with an open InnoDB transaction that
// IS currently running a query (trx_query IS NOT NULL) — the inverse of
// idleInTransactionQuery's filter, which only differs by that one
// condition. Query text comes from events_statements_current.DIGEST_TEXT
// (parameter values replaced with ?), never from innodb_trx.trx_query
// directly — trx_query is MySQL's raw, unnormalized text, and using it
// would repeat the exact literal-value exposure bug already found and
// fixed in the Postgres adapter's equivalent collector. Owner comes from
// a left join to performance_schema.threads, same as the
// idle-in-transaction adapter.
const longRunningQueryQuery = `
SELECT
    t.trx_mysql_thread_id,
    COALESCE(th.PROCESSLIST_USER, ''),
    COALESCE(th.PROCESSLIST_DB, ''),
    COALESCE(esc.DIGEST_TEXT, '[query not tracked]'),
    TIMESTAMPDIFF(SECOND, t.trx_started, NOW())
FROM information_schema.innodb_trx t
LEFT JOIN performance_schema.threads th ON th.PROCESSLIST_ID = t.trx_mysql_thread_id
LEFT JOIN performance_schema.events_statements_current esc ON esc.THREAD_ID = th.THREAD_ID
WHERE t.trx_state = 'RUNNING' AND t.trx_query IS NOT NULL
`

// LongRunningQueryCollector reads sessions currently executing a query
// inside an open InnoDB transaction, from information_schema.innodb_trx.
// It has no opinion on how long is "too long" — that judgment belongs to
// domain.DetectLongRunningQueryWarnings, unchanged from the Postgres
// adapter.
type LongRunningQueryCollector struct {
	db *sql.DB
}

func NewLongRunningQueryCollector(db *sql.DB) *LongRunningQueryCollector {
	return &LongRunningQueryCollector{db: db}
}

func (c *LongRunningQueryCollector) Fetch(ctx context.Context) ([]domain.LongRunningQuerySession, error) {
	rows, err := c.db.QueryContext(ctx, longRunningQueryQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	sessions := make([]domain.LongRunningQuerySession, 0)
	for rows.Next() {
		var threadID int64
		var s domain.LongRunningQuerySession
		if err := rows.Scan(&threadID, &s.User, &s.ApplicationName, &s.Query, &s.RunningSeconds); err != nil {
			return nil, err
		}
		s.PID = int32(threadID)
		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}
