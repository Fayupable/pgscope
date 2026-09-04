package mysql

import "github.com/fayupable/pgscope/internal/domain"

// classifyLockSeverity maps MySQL's data_locks LOCK_TYPE/LOCK_MODE pair to
// the engine-agnostic domain.LockSeverity, this adapter's own version of
// what the Postgres adapter's classifyLockSeverity does for Postgres's lock
// vocabulary (see collector.go's comment there — this is exactly the
// MySQL-specific mapping it anticipated).
//
// LOCK_TYPE matters as much as LOCK_MODE here: MySQL's 'X' mode means two
// very different things depending on scope. A RECORD-level X lock (the
// ordinary per-row lock any UPDATE/DELETE/SELECT..FOR UPDATE takes) only
// conflicts with another lock on that same row — functionally equivalent
// to Postgres's RowExclusiveLock (SharedWrite), not a big deal, multiple
// sessions writing different rows proceed fine. A TABLE-level X lock (from
// DDL, LOCK TABLES, or a full-table scan needing to lock everything) blocks
// the entire table — that's Postgres's AccessExclusiveLock (Exclusive).
// LOCK_MODE also carries GAP-lock suffixes (e.g. "X,GAP", "X,REC_NOT_GAP")
// that don't change this classification — only the base mode before the
// comma does.
func classifyLockSeverity(lockType, lockMode string) domain.LockSeverity {
	base := baseLockMode(lockMode)

	switch base {
	case "S":
		return domain.LockSeveritySharedRead
	case "X":
		if lockType == "TABLE" {
			return domain.LockSeverityExclusive
		}
		return domain.LockSeveritySharedWrite
	case "IS", "IX":
		return domain.LockSeverityIntent
	default:
		return domain.LockSeverityUnknown
	}
}

func baseLockMode(lockMode string) string {
	for i := 0; i < len(lockMode); i++ {
		if lockMode[i] == ',' {
			return lockMode[:i]
		}
	}
	return lockMode
}
