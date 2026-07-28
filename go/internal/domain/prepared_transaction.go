package domain

import "fmt"

const PreparedTransactionWarningSeconds = 600.0

// PreparedTransactionInfo is the raw shape read from pg_prepared_xacts for
// one two-phase-commit transaction still awaiting COMMIT PREPARED or
// ROLLBACK PREPARED — no judgment applied yet.
type PreparedTransactionInfo struct {
	GID        string
	Database   string
	Owner      string
	AgeSeconds float64
}

// PreparedTransactionWarning is a suggestion, never a certainty —
// pg_prepared_xacts is expected to be empty almost all the time outside the
// brief window a two-phase commit coordinator is actively finishing up. One
// sitting there a while usually means the coordinator crashed or forgot to
// follow up, and it holds locks and blocks vacuum for as long as it stays
// open.
type PreparedTransactionWarning struct {
	GID         string  `json:"gid"`
	Database    string  `json:"database"`
	Owner       string  `json:"owner"`
	AgeSeconds  float64 `json:"ageSeconds"`
	Explanation string  `json:"explanation"`
}

// DetectPreparedTransactionWarnings filters prepared transactions down to
// the ones open long enough to matter — a transaction mid-commit for a
// second or two is a coordinator doing its job, not a signal.
func DetectPreparedTransactionWarnings(transactions []PreparedTransactionInfo) []PreparedTransactionWarning {
	result := make([]PreparedTransactionWarning, 0)
	for _, tx := range transactions {
		if tx.AgeSeconds < PreparedTransactionWarningSeconds {
			continue
		}

		result = append(result, PreparedTransactionWarning{
			GID:        tx.GID,
			Database:   tx.Database,
			Owner:      tx.Owner,
			AgeSeconds: tx.AgeSeconds,
			Explanation: fmt.Sprintf(
				"Prepared transaction %q on database %q (owner %q) has been waiting %.0f seconds to be committed or rolled back. It's holding locks and blocking vacuum for as long as it stays open — likely a crashed or forgotten two-phase commit coordinator. Run COMMIT PREPARED or ROLLBACK PREPARED to resolve it.",
				tx.GID, tx.Database, tx.Owner, tx.AgeSeconds,
			),
		})
	}
	return result
}
