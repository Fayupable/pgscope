package domain

import "time"

type SessionState string

const (
	SessionStateActive            SessionState = "active"
	SessionStateIdle              SessionState = "idle"
	SessionStateIdleInTransaction SessionState = "idle_in_transaction"
	SessionStateDisabled          SessionState = "disabled"
)

type WaitEventType string

const (
	WaitEventTypeNone    WaitEventType = "none"
	WaitEventTypeLock    WaitEventType = "lock"
	WaitEventTypeIO      WaitEventType = "io"
	WaitEventTypeCPU     WaitEventType = "cpu"
	WaitEventTypeClient  WaitEventType = "client"
	WaitEventTypeIPC     WaitEventType = "ipc"
	WaitEventTypeTimeout WaitEventType = "timeout"
)

// LockSeverity is the normalized, engine-agnostic category every database's
// native lock vocabulary maps down to. This is the only lock classification
// the domain layer and frontend reason about — never a specific engine's
// lock mode name.
type LockSeverity string

const (
	LockSeverityUnknown     LockSeverity = "unknown"
	LockSeveritySharedRead  LockSeverity = "shared_read"
	LockSeveritySharedWrite LockSeverity = "shared_write"
	LockSeverityIntent      LockSeverity = "intent"
	LockSeverityExclusive   LockSeverity = "exclusive"
)

// QueryOperation classifies the statement a session is running, derived from
// its SQL command's leading keyword. Identical across every SQL engine.
type QueryOperation string

const (
	QueryOperationSelect  QueryOperation = "SELECT"
	QueryOperationInsert  QueryOperation = "INSERT"
	QueryOperationUpdate  QueryOperation = "UPDATE"
	QueryOperationDelete  QueryOperation = "DELETE"
	QueryOperationMerge   QueryOperation = "MERGE"
	QueryOperationDDL     QueryOperation = "DDL"
	QueryOperationVacuum  QueryOperation = "VACUUM"
	QueryOperationOther   QueryOperation = "OTHER"
	QueryOperationUnknown QueryOperation = "UNKNOWN"
)

// LockedObject describes one lock entry held or awaited by a session.
// NativeMode carries the engine's own lock name verbatim (e.g.
// "AccessExclusiveLock" on Postgres, "X" on MySQL) purely for display and
// debugging — no domain logic may branch on it. Severity is the normalized
// value domain logic and the frontend actually use; each engine adapter is
// responsible for mapping its native mode to a Severity.
type LockedObject struct {
	NativeMode string       `json:"nativeMode"`
	Severity   LockSeverity `json:"severity"`
	Resource   string       `json:"resource,omitempty"`
	Granted    bool         `json:"granted"`
}

// Session represents one active backend/connection on the target database,
// independent of which underlying engine (Postgres, MySQL, ...) produced it.
type Session struct {
	ID              string
	User            string
	ApplicationName string
	ClientAddress   string
	State           SessionState
	WaitEventType   WaitEventType
	WaitEvent       string
	Query           string
	Operation       QueryOperation
	QueryStarted    time.Time
	Duration        time.Duration
	BlockedBy       []string
	Locks           []LockedObject
}

func (s Session) IsBlocked() bool {
	return len(s.BlockedBy) > 0
}
