package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const activeSessionsQuery = `
SELECT
    a.pid,
    COALESCE(a.usename, ''),
    COALESCE(a.application_name, ''),
    COALESCE(a.client_addr::text, ''),
    COALESCE(a.state, ''),
    COALESCE(a.wait_event_type, ''),
    COALESCE(a.wait_event, ''),
    COALESCE(s.query, '[query not tracked by pg_stat_statements]'),
    COALESCE(a.query_start, now()),
    pg_blocking_pids(a.pid)
FROM pg_stat_activity a
LEFT JOIN pg_stat_statements s ON s.queryid = a.query_id
WHERE a.pid != pg_backend_pid()
  AND a.state IS NOT NULL
  AND a.state != 'idle'
`

const sessionLocksQuery = `
SELECT
    pid,
    locktype,
    COALESCE(relation::regclass::text, ''),
    mode,
    granted
FROM pg_locks
WHERE pid = ANY($1)
  AND locktype NOT IN ('virtualxid', 'transactionid')
`

// Collector implements output.ISessionCollectorPort against PostgreSQL's
// pg_stat_activity, pg_stat_statements, and pg_locks system views. It never
// mutates data; the connection pool it receives is expected to already
// enforce read-only + timeouts at the role/session level.
//
// Query text always comes from pg_stat_statements (parameter values replaced
// with $1, $2, ...), never from pg_stat_activity's raw query column — this
// prevents literal values (passwords, PII, tokens) that appear in a running
// statement from ever being exposed through the tool.
type Collector struct {
	pool *pgxpool.Pool
}

func NewCollector(pool *pgxpool.Pool) *Collector {
	return &Collector{pool: pool}
}

func (c *Collector) FetchActiveSessions(ctx context.Context) ([]domain.Session, error) {
	sessions, err := c.fetchSessions(ctx)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return sessions, nil
	}

	locksByPID, err := c.fetchLocksByPID(ctx, sessionPIDs(sessions))
	if err != nil {
		return nil, err
	}

	for i := range sessions {
		locks, ok := locksByPID[sessions[i].ID]
		if !ok {
			locks = make([]domain.LockedObject, 0)
		}
		sessions[i].Locks = locks
	}

	return sessions, nil
}

func (c *Collector) fetchSessions(ctx context.Context) ([]domain.Session, error) {
	rows, err := c.pool.Query(ctx, activeSessionsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]domain.Session, 0)
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func (c *Collector) fetchLocksByPID(ctx context.Context, pids []int32) (map[string][]domain.LockedObject, error) {
	rows, err := c.pool.Query(ctx, sessionLocksQuery, pids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locksByPID := make(map[string][]domain.LockedObject)
	for rows.Next() {
		pid, lock, err := scanLock(rows)
		if err != nil {
			return nil, err
		}
		locksByPID[pid] = append(locksByPID[pid], lock)
	}

	return locksByPID, rows.Err()
}

func scanSession(rows pgx.Rows) (domain.Session, error) {
	var (
		pid             int32
		user            string
		applicationName string
		clientAddress   string
		state           string
		waitEventType   string
		waitEvent       string
		query           string
		queryStarted    time.Time
		blockingPids    []int32
	)

	if err := rows.Scan(
		&pid, &user, &applicationName, &clientAddress,
		&state, &waitEventType, &waitEvent,
		&query, &queryStarted, &blockingPids,
	); err != nil {
		return domain.Session{}, err
	}

	return domain.Session{
		ID:              strconv.Itoa(int(pid)),
		User:            user,
		ApplicationName: applicationName,
		ClientAddress:   clientAddress,
		State:           domain.SessionState(state),
		WaitEventType:   domain.WaitEventType(waitEventType),
		WaitEvent:       waitEvent,
		Query:           query,
		Operation:       domain.ClassifyOperation(query),
		QueryStarted:    queryStarted,
		Duration:        time.Since(queryStarted),
		BlockedBy:       intsToStrings(blockingPids),
	}, nil
}

func scanLock(rows pgx.Rows) (string, domain.LockedObject, error) {
	var (
		pid      int32
		lockType string
		relation string
		mode     string
		granted  bool
	)

	if err := rows.Scan(&pid, &lockType, &relation, &mode, &granted); err != nil {
		return "", domain.LockedObject{}, err
	}

	return strconv.Itoa(int(pid)), domain.LockedObject{
		NativeMode: mode,
		Severity:   classifyLockSeverity(mode),
		Resource:   relation,
		Granted:    granted,
	}, nil
}

// classifyLockSeverity maps Postgres's native lock mode names to the
// engine-agnostic domain.LockSeverity. This mapping is Postgres-specific by
// design — a MySQL adapter would implement its own version against MySQL's
// own lock vocabulary (S/X/IS/IX/gap locks).
func classifyLockSeverity(nativeMode string) domain.LockSeverity {
	switch nativeMode {
	case "AccessShareLock":
		return domain.LockSeveritySharedRead
	case "RowShareLock", "RowExclusiveLock":
		return domain.LockSeveritySharedWrite
	case "ShareUpdateExclusiveLock", "ShareLock", "ShareRowExclusiveLock":
		return domain.LockSeverityIntent
	case "ExclusiveLock", "AccessExclusiveLock":
		return domain.LockSeverityExclusive
	default:
		return domain.LockSeverityUnknown
	}
}

func sessionPIDs(sessions []domain.Session) []int32 {
	pids := make([]int32, len(sessions))
	for i, s := range sessions {
		pid, _ := strconv.Atoi(s.ID)
		pids[i] = int32(pid)
	}
	return pids
}

func intsToStrings(values []int32) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = strconv.Itoa(int(v))
	}
	return result
}
