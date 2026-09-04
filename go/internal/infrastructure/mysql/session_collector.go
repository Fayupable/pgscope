package mysql

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/fayupable/pgscope/internal/domain"
)

// activeThreadsQuery reads every connection's current activity from
// performance_schema.threads — MySQL's equivalent of pg_stat_activity.
// Query text comes from events_statements_current.DIGEST_TEXT (normalized,
// literal values replaced with ?), never from threads.PROCESSLIST_INFO,
// which carries the raw, unmasked SQL — same reasoning as every other
// collector that reads query text in this package.
//
// performance_schema.threads also lists MySQL's own background threads
// (event_scheduler, compress_gtid_table, replication I/O/SQL threads, ...)
// with PROCESSLIST_COMMAND = 'Daemon' or similar — not real client
// sessions, and never something a user would want listed as one.
// PROCESSLIST_COMMAND is restricted to ('Query', 'Sleep') to exclude them,
// verified empirically against a live server (see session log for
// 2026-08-29): two Daemon threads slipped through an earlier, looser
// "!= 'Sleep'" filter before this fix.
//
// Unlike Postgres, MySQL has no single "idle in transaction" state on this
// view: a real client connection's PROCESSLIST_COMMAND only ever says
// 'Query' or 'Sleep', regardless of whether it's sitting inside an open,
// uncommitted transaction. innodb_trx is joined in separately (same source
// idle_in_transaction_collector.go already reads) so sessionState can tell
// the two apart.
const activeThreadsQuery = `
SELECT
    th.PROCESSLIST_ID,
    COALESCE(th.PROCESSLIST_USER, ''),
    COALESCE(th.PROCESSLIST_DB, ''),
    COALESCE(th.PROCESSLIST_HOST, ''),
    th.PROCESSLIST_COMMAND,
    COALESCE(esc.DIGEST_TEXT, '[query not tracked]'),
    COALESCE(th.PROCESSLIST_TIME, 0),
    (t.trx_id IS NOT NULL) AS has_open_trx
FROM performance_schema.threads th
LEFT JOIN performance_schema.events_statements_current esc ON esc.THREAD_ID = th.THREAD_ID
LEFT JOIN information_schema.innodb_trx t ON t.trx_mysql_thread_id = th.PROCESSLIST_ID
WHERE th.PROCESSLIST_ID IS NOT NULL
  AND th.PROCESSLIST_ID != CONNECTION_ID()
  AND th.PROCESSLIST_COMMAND IN ('Query', 'Sleep')
  AND (th.PROCESSLIST_COMMAND != 'Sleep' OR t.trx_id IS NOT NULL)
`

// blockingPidsQuery reads just the waiting-pid/blocking-pid relationship
// from sys.innodb_lock_waits — the same view LockWaitCollector reads, but
// without its query-text joins, since BlockedBy only needs PIDs. A waiting
// session can appear more than once here if it's waiting on locks held by
// more than one blocker.
const blockingPidsQuery = `SELECT waiting_pid, blocking_pid FROM sys.innodb_lock_waits`

// sessionLocksQuery reads every lock a session currently holds or is
// waiting on, from performance_schema.data_locks — MySQL's equivalent of
// pg_locks. Joined against threads to translate its internal THREAD_ID
// into the PROCESSLIST_ID (pid) domain.Session is keyed by.
const sessionLocksQuery = `
SELECT
    th.PROCESSLIST_ID,
    dl.LOCK_TYPE,
    dl.LOCK_MODE,
    COALESCE(dl.OBJECT_NAME, ''),
    (dl.LOCK_STATUS = 'GRANTED')
FROM performance_schema.data_locks dl
JOIN performance_schema.threads th ON th.THREAD_ID = dl.THREAD_ID
WHERE th.PROCESSLIST_ID IS NOT NULL
`

// SessionCollector reads active connections from performance_schema, plus
// (via sys.innodb_lock_waits) which sessions are currently blocked waiting
// for a lock another session holds, and (via performance_schema.data_locks)
// every lock each session holds or is waiting on — the same three pieces
// the Postgres adapter's Collector assembles from pg_stat_activity,
// pg_blocking_pids(), and pg_locks.
type SessionCollector struct {
	db *sql.DB
}

func NewSessionCollector(db *sql.DB) *SessionCollector {
	return &SessionCollector{db: db}
}

func (c *SessionCollector) FetchActiveSessions(ctx context.Context) ([]domain.Session, error) {
	rows, err := c.db.QueryContext(ctx, activeThreadsQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	sessions := make([]domain.Session, 0)
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return sessions, nil
	}

	blockedBy, err := c.fetchBlockingPIDs(ctx)
	if err != nil {
		return nil, err
	}
	locksByPID, err := c.fetchLocksByPID(ctx)
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		// blockedBy[id] is nil (not an empty slice) for any session absent
		// from the map — scanSession already defaults BlockedBy to an
		// empty, non-nil slice, so only overwrite it when a real match
		// exists, keeping JSON output "[]" rather than "null" either way.
		if pids, found := blockedBy[sessions[i].ID]; found {
			sessions[i].BlockedBy = pids
		}
		if locks, found := locksByPID[sessions[i].ID]; found {
			sessions[i].Locks = locks
		}
	}

	return sessions, nil
}

func (c *SessionCollector) fetchLocksByPID(ctx context.Context) (map[string][]domain.LockedObject, error) {
	rows, err := c.db.QueryContext(ctx, sessionLocksQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	locksByPID := make(map[string][]domain.LockedObject)
	for rows.Next() {
		var pid int64
		var lockType, lockMode, resource string
		var granted bool
		if err := rows.Scan(&pid, &lockType, &lockMode, &resource, &granted); err != nil {
			return nil, err
		}

		id := strconv.FormatInt(pid, 10)
		locksByPID[id] = append(locksByPID[id], domain.LockedObject{
			NativeMode: lockMode,
			Severity:   classifyLockSeverity(lockType, lockMode),
			Resource:   resource,
			Granted:    granted,
		})
	}

	return locksByPID, rows.Err()
}

func (c *SessionCollector) fetchBlockingPIDs(ctx context.Context) (map[string][]string, error) {
	rows, err := c.db.QueryContext(ctx, blockingPidsQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	blockedBy := make(map[string][]string)
	for rows.Next() {
		var waitingPID, blockingPID int64
		if err := rows.Scan(&waitingPID, &blockingPID); err != nil {
			return nil, err
		}
		waiting := strconv.FormatInt(waitingPID, 10)
		blockedBy[waiting] = append(blockedBy[waiting], strconv.FormatInt(blockingPID, 10))
	}

	return blockedBy, rows.Err()
}

func scanSession(rows *sql.Rows) (domain.Session, error) {
	var (
		pid             int64
		user            string
		db              string
		host            string
		command         string
		query           string
		processlistTime int64
		hasOpenTxn      bool
	)

	if err := rows.Scan(&pid, &user, &db, &host, &command, &query, &processlistTime, &hasOpenTxn); err != nil {
		return domain.Session{}, err
	}

	queryStarted := time.Now().Add(-time.Duration(processlistTime) * time.Second)

	return domain.Session{
		ID:              strconv.FormatInt(pid, 10),
		User:            user,
		ApplicationName: db,
		ClientAddress:   host,
		State:           sessionState(command, hasOpenTxn),
		WaitEventType:   domain.WaitEventTypeNone,
		Query:           query,
		Operation:       domain.ClassifyOperation(query),
		QueryStarted:    queryStarted,
		Duration:        time.Duration(processlistTime) * time.Second,
		BlockedBy:       make([]string, 0),
		Locks:           make([]domain.LockedObject, 0),
	}, nil
}

// sessionState maps MySQL's coarse PROCESSLIST_COMMAND ('Query' or
// 'Sleep') plus whether an InnoDB transaction is currently open into
// domain.SessionState's three-way distinction. A 'Sleep' connection with
// no open transaction is genuinely idle and excluded by activeThreadsQuery
// before this ever runs — every row reaching here is either running a
// query or idling inside an open transaction.
func sessionState(command string, hasOpenTxn bool) domain.SessionState {
	if command == "Query" {
		return domain.SessionStateActive
	}
	if hasOpenTxn {
		return domain.SessionStateIdleInTransaction
	}
	return domain.SessionStateIdle
}
