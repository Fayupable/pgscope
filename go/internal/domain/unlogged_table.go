package domain

import "fmt"

// UnloggedTable is a table created with UNLOGGED — Postgres skips
// write-ahead logging for it, which makes writes faster but means the
// table is not crash-safe: it's silently truncated after an unclean
// shutdown (a crash, an unexpected power loss, ...). Existing simply
// because the table is unlogged is the finding here — there's no threshold
// to cross, just a fact worth surfacing.
type UnloggedTable struct {
	Table       string `json:"table"`
	Explanation string `json:"explanation"`
}

func NewUnloggedTable(table string) UnloggedTable {
	return UnloggedTable{
		Table: table,
		Explanation: fmt.Sprintf(
			"Table %q is UNLOGGED. Writes to it skip the write-ahead log, which makes them faster, but the table is silently truncated (all rows lost) after an unclean shutdown (a crash, power loss, or `pg_ctl stop -m immediate`). Confirm this is intentional — if this table holds anything you'd need after a crash, consider making it a regular (logged) table instead.",
			table,
		),
	}
}
