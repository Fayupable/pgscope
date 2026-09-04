package mysql

import (
	"context"
	"database/sql"

	"github.com/fayupable/pgscope/internal/domain"
)

// lockWaitQuery reads the blocking relationship from sys.innodb_lock_waits
// — who is waiting, who is blocking, on which table, for how long. Query
// text deliberately does NOT come from sys.innodb_lock_waits's own
// waiting_query/blocking_query columns: those carry the raw, unnormalized
// SQL text (literal values included), the same class of exposure already
// found and fixed in long_running_query_collector.go. Instead, both are
// joined by PID through performance_schema.threads to
// events_statements_current.DIGEST_TEXT, MySQL's normalized ($N-equivalent)
// query text.
const lockWaitQuery = `
SELECT
    w.waiting_pid,
    COALESCE(wesc.DIGEST_TEXT, '[query not tracked]'),
    w.blocking_pid,
    COALESCE(besc.DIGEST_TEXT, '[query not tracked]'),
    w.locked_table_name,
    w.wait_age_secs
FROM sys.innodb_lock_waits w
LEFT JOIN performance_schema.threads wth ON wth.PROCESSLIST_ID = w.waiting_pid
LEFT JOIN performance_schema.events_statements_current wesc ON wesc.THREAD_ID = wth.THREAD_ID
LEFT JOIN performance_schema.threads bth ON bth.PROCESSLIST_ID = w.blocking_pid
LEFT JOIN performance_schema.events_statements_current besc ON besc.THREAD_ID = bth.THREAD_ID
`

// LockWaitCollector reads InnoDB row-lock wait relationships from the sys
// schema. It has no opinion on how long a wait is "too long" — that
// judgment belongs to domain.DetectLockWaitWarnings.
type LockWaitCollector struct {
	db *sql.DB
}

func NewLockWaitCollector(db *sql.DB) *LockWaitCollector {
	return &LockWaitCollector{db: db}
}

func (c *LockWaitCollector) Fetch(ctx context.Context) ([]domain.LockWaitSession, error) {
	rows, err := c.db.QueryContext(ctx, lockWaitQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	sessions := make([]domain.LockWaitSession, 0)
	for rows.Next() {
		var waitingPID, blockingPID int64
		var s domain.LockWaitSession
		if err := rows.Scan(&waitingPID, &s.WaitingQuery, &blockingPID, &s.BlockingQuery, &s.LockedTable, &s.WaitAgeSeconds); err != nil {
			return nil, err
		}
		s.WaitingPID = int32(waitingPID)
		s.BlockingPID = int32(blockingPID)
		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}
