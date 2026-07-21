package domain

import "fmt"

const IdleInTransactionWarningSeconds = 60.0

// IdleInTransactionSession is the raw shape read from pg_stat_activity
// for one session sitting idle inside an open transaction — no
// judgment applied yet.
type IdleInTransactionSession struct {
	PID             int32
	User            string
	ApplicationName string
	IdleSeconds     float64
}

// IdleInTransactionWarning is a suggestion, never a certainty — a
// session idle in a transaction is still holding whatever locks and
// snapshot it acquired, which can block vacuum and other queries the
// longer it sits there.
type IdleInTransactionWarning struct {
	PID             int32   `json:"pid"`
	User            string  `json:"user"`
	ApplicationName string  `json:"applicationName"`
	IdleSeconds     float64 `json:"idleSeconds"`
	Explanation     string  `json:"explanation"`
}

// DetectIdleInTransactionWarnings filters sessions down to the ones
// idle long enough to matter — a session idle for a second or two
// mid-request is normal, not a signal.
func DetectIdleInTransactionWarnings(sessions []IdleInTransactionSession) []IdleInTransactionWarning {
	result := make([]IdleInTransactionWarning, 0)
	for _, s := range sessions {
		if s.IdleSeconds < IdleInTransactionWarningSeconds {
			continue
		}

		result = append(result, IdleInTransactionWarning{
			PID:             s.PID,
			User:            s.User,
			ApplicationName: s.ApplicationName,
			IdleSeconds:     s.IdleSeconds,
			Explanation: fmt.Sprintf(
				"Session %d (user %q, app %q) has been idle in a transaction for %.0f seconds. It's still holding any locks and snapshot it acquired — this can block vacuum and other queries. Consider setting idle_in_transaction_session_timeout, or check why this client isn't committing/rolling back.",
				s.PID, s.User, s.ApplicationName, s.IdleSeconds,
			),
		})
	}
	return result
}
