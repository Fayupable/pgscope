package domain

import "fmt"

const LongRunningQueryWarningSeconds = 300.0

// LongRunningQuerySession is the raw shape read from pg_stat_activity for
// one session with state = 'active' — no judgment applied yet. Unlike
// IdleInTransactionSession, this session is actually executing a query
// right now, not waiting on its client.
type LongRunningQuerySession struct {
	PID             int32
	User            string
	ApplicationName string
	Query           string
	RunningSeconds  float64
}

// LongRunningQueryWarning is a suggestion, never a certainty — some
// queries (large reports, batch jobs, maintenance tasks) are legitimately
// long-running by design. A query stuck in 'active' for a while without
// waiting on a lock usually signals a plan gone bad (e.g. a nested loop
// over huge tables), worth a look either way.
type LongRunningQueryWarning struct {
	PID             int32   `json:"pid"`
	User            string  `json:"user"`
	ApplicationName string  `json:"applicationName"`
	Query           string  `json:"query"`
	RunningSeconds  float64 `json:"runningSeconds"`
	Explanation     string  `json:"explanation"`
}

// DetectLongRunningQueryWarnings filters active sessions down to the ones
// running long enough to matter — most queries finish in milliseconds, so
// this floor is set high enough to only flag genuine outliers.
func DetectLongRunningQueryWarnings(sessions []LongRunningQuerySession) []LongRunningQueryWarning {
	result := make([]LongRunningQueryWarning, 0)
	for _, s := range sessions {
		if s.RunningSeconds < LongRunningQueryWarningSeconds {
			continue
		}

		result = append(result, LongRunningQueryWarning{
			PID:             s.PID,
			User:            s.User,
			ApplicationName: s.ApplicationName,
			Query:           s.Query,
			RunningSeconds:  s.RunningSeconds,
			Explanation: fmt.Sprintf(
				"Session %d (user %q, app %q) has been actively running the same query for %.0f seconds. This may be an expected long-running job (a report, batch job, or maintenance task), but if it's unexpected, it often signals a bad query plan (e.g. a sequential or nested-loop scan over a much larger table than intended). The query text shown here is normalized (literal values replaced with $1, $2, ...) — to see the real query with its actual values, connect to the database yourself and run: SELECT query, wait_event_type, wait_event FROM pg_stat_activity WHERE pid = %d.",
				s.PID, s.User, s.ApplicationName, s.RunningSeconds, s.PID,
			),
		})
	}
	return result
}
