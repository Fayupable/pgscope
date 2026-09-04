package domain

import "fmt"

const LockWaitWarningSeconds = 5.0

// LockWaitSession is one session currently blocked waiting to acquire a
// row lock another session already holds — no judgment applied yet. This
// has no direct Postgres equivalent in pgscope today: Postgres exposes
// lock waits through pg_locks joined against pg_stat_activity, which
// pgscope doesn't yet surface as its own insight (that's part of the
// still-unbuilt live session/lock graph). MySQL's sys.innodb_lock_waits
// view makes the same relationship — who is blocking whom — available as
// a simple read, so this ships as a MySQL-only insight for now.
type LockWaitSession struct {
	WaitingPID     int32
	WaitingQuery   string
	BlockingPID    int32
	BlockingQuery  string
	LockedTable    string
	WaitAgeSeconds float64
}

// LockWaitWarning is a suggestion, never a certainty — brief lock waits
// (a few milliseconds to a couple seconds) are completely normal in any
// database under concurrent write load. One stretching well beyond that
// usually means a transaction is holding a lock much longer than it
// needs to (an idle-in-transaction session, a slow client, a forgotten
// COMMIT), and every other session waiting behind it is now stalled too.
type LockWaitWarning struct {
	WaitingPID     int32   `json:"waitingPid"`
	WaitingQuery   string  `json:"waitingQuery"`
	BlockingPID    int32   `json:"blockingPid"`
	BlockingQuery  string  `json:"blockingQuery"`
	LockedTable    string  `json:"lockedTable"`
	WaitAgeSeconds float64 `json:"waitAgeSeconds"`
	Explanation    string  `json:"explanation"`
}

// DetectLockWaitWarnings filters lock waits down to the ones stretching
// long enough to matter — most lock waits resolve in well under a
// second and are just normal contention, not a signal.
func DetectLockWaitWarnings(sessions []LockWaitSession) []LockWaitWarning {
	result := make([]LockWaitWarning, 0)
	for _, s := range sessions {
		if s.WaitAgeSeconds < LockWaitWarningSeconds {
			continue
		}

		result = append(result, LockWaitWarning{
			WaitingPID:     s.WaitingPID,
			WaitingQuery:   s.WaitingQuery,
			BlockingPID:    s.BlockingPID,
			BlockingQuery:  s.BlockingQuery,
			LockedTable:    s.LockedTable,
			WaitAgeSeconds: s.WaitAgeSeconds,
			Explanation: fmt.Sprintf(
				"Session %d has been waiting %.1f seconds for a lock on table %q held by session %d. If session %d isn't about to commit or roll back on its own, this is worth investigating — every session queued behind it is stalled for as long as it stays open.",
				s.WaitingPID, s.WaitAgeSeconds, s.LockedTable, s.BlockingPID, s.BlockingPID,
			),
		})
	}
	return result
}
