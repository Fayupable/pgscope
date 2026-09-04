package mysql

import (
	"context"
	"database/sql"

	"github.com/fayupable/pgscope/internal/domain"
)

// idleInTransactionQuery reads sessions with an open InnoDB transaction
// that isn't currently running a query (trx_query IS NULL) — MySQL's
// closest equivalent to Postgres's state = 'idle in transaction'. Owner
// and application come from a left join to performance_schema.threads,
// since innodb_trx alone doesn't carry the connecting user (MySQL has no
// application_name equivalent, so the connecting database is used there
// instead — still useful for identifying which client left this open).
const idleInTransactionQuery = `
SELECT
    t.trx_mysql_thread_id,
    COALESCE(th.PROCESSLIST_USER, ''),
    COALESCE(th.PROCESSLIST_DB, ''),
    TIMESTAMPDIFF(SECOND, t.trx_started, NOW())
FROM information_schema.innodb_trx t
LEFT JOIN performance_schema.threads th ON th.PROCESSLIST_ID = t.trx_mysql_thread_id
WHERE t.trx_state = 'RUNNING' AND t.trx_query IS NULL
`

// IdleInTransactionCollector reads open-but-idle InnoDB transactions from
// information_schema.innodb_trx. It has no opinion on how long is "too
// long" — that judgment belongs to domain.DetectIdleInTransactionWarnings,
// unchanged from the Postgres adapter.
type IdleInTransactionCollector struct {
	db *sql.DB
}

func NewIdleInTransactionCollector(db *sql.DB) *IdleInTransactionCollector {
	return &IdleInTransactionCollector{db: db}
}

func (c *IdleInTransactionCollector) Fetch(ctx context.Context) ([]domain.IdleInTransactionSession, error) {
	rows, err := c.db.QueryContext(ctx, idleInTransactionQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	sessions := make([]domain.IdleInTransactionSession, 0)
	for rows.Next() {
		var threadID int64
		var s domain.IdleInTransactionSession
		if err := rows.Scan(&threadID, &s.User, &s.ApplicationName, &s.IdleSeconds); err != nil {
			return nil, err
		}
		s.PID = int32(threadID)
		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}
