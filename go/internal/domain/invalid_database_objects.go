package domain

import "fmt"

// InvalidIndex is a leftover from a failed or interrupted CREATE INDEX
// CONCURRENTLY — it exists in the catalog but the database won't use it
// for anything. It occupies disk space and does nothing.
type InvalidIndex struct {
	Table       string `json:"table"`
	Index       string `json:"index"`
	Explanation string `json:"explanation"`
}

// UnvalidatedConstraint is a constraint added with NOT VALID (or still
// mid-validation) — Postgres isn't enforcing it against existing rows
// yet, only against new writes.
type UnvalidatedConstraint struct {
	Table       string `json:"table"`
	Constraint  string `json:"constraint"`
	Explanation string `json:"explanation"`
}

func NewInvalidIndex(table, index string) InvalidIndex {
	return InvalidIndex{
		Table: table,
		Index: index,
		Explanation: fmt.Sprintf(
			"Index %q on %s is marked invalid — likely a CREATE INDEX CONCURRENTLY that failed or was interrupted. It takes up disk space but Postgres won't use it. Consider DROP INDEX and rebuilding it.",
			index, table,
		),
	}
}

func NewUnvalidatedConstraint(table, constraint string) UnvalidatedConstraint {
	return UnvalidatedConstraint{
		Table:      table,
		Constraint: constraint,
		Explanation: fmt.Sprintf(
			"Constraint %q on %s is not validated — Postgres enforces it for new writes but hasn't confirmed existing rows satisfy it. Run ALTER TABLE ... VALIDATE CONSTRAINT to confirm and enable it fully.",
			constraint, table,
		),
	}
}
