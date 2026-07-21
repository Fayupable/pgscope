package domain

import (
	"fmt"
	"strings"
)

// IndexInfo describes one simple (non-expression) index as read directly
// from the system catalog — no text parsing involved.
type IndexInfo struct {
	Table        string
	Name         string
	AccessMethod string
	Unique       bool
	Primary      bool
	Partial      bool
	Predicate    string
	Columns      []string
}

// DuplicateIndex is a suggestion, never a certainty — same philosophy as
// IndexCandidate. It always names both indexes involved so the caller can
// verify the reasoning themselves before dropping anything.
type DuplicateIndex struct {
	Table            string   `json:"table"`
	RedundantIndex   string   `json:"redundantIndex"`
	CoveringIndex    string   `json:"coveringIndex"`
	RedundantColumns []string `json:"redundantColumns"`
	CoveringColumns  []string `json:"coveringColumns"`
	Explanation      string   `json:"explanation"`
}

// DetectDuplicateIndexes finds indexes made redundant by another index on
// the same table that already covers everything the redundant one does.
// An index B "covers" index A when B's leading columns are exactly A's
// columns, in the same order, using the same access method and the same
// partial-index predicate (if any) — meaning every lookup A could serve,
// B could serve too. Primary keys and unique constraints are never
// flagged as redundant themselves, since dropping them changes semantics,
// not just performance.
func DetectDuplicateIndexes(indexes []IndexInfo) []DuplicateIndex {
	byTable := make(map[string][]IndexInfo)
	for _, idx := range indexes {
		byTable[idx.Table] = append(byTable[idx.Table], idx)
	}

	duplicates := make([]DuplicateIndex, 0)
	for table, tableIndexes := range byTable {
		duplicates = append(duplicates, duplicatesWithinTable(table, tableIndexes)...)
	}

	return duplicates
}

func duplicatesWithinTable(table string, indexes []IndexInfo) []DuplicateIndex {
	duplicates := make([]DuplicateIndex, 0)

	for _, candidate := range indexes {
		if candidate.Primary || candidate.Unique {
			continue
		}

		covering, found := findCoveringIndex(candidate, indexes)
		if !found {
			continue
		}

		duplicates = append(duplicates, DuplicateIndex{
			Table:            table,
			RedundantIndex:   candidate.Name,
			CoveringIndex:    covering.Name,
			RedundantColumns: candidate.Columns,
			CoveringColumns:  covering.Columns,
			Explanation:      buildDuplicateExplanation(candidate, covering),
		})
	}

	return duplicates
}

func findCoveringIndex(candidate IndexInfo, siblings []IndexInfo) (IndexInfo, bool) {
	for _, other := range siblings {
		if other.Name == candidate.Name {
			continue
		}
		if !sameShape(candidate, other) {
			continue
		}
		if !columnsCovered(other.Columns, candidate.Columns) {
			continue
		}
		if preferOther(candidate, other) {
			return other, true
		}
	}
	return IndexInfo{}, false
}

func sameShape(a, b IndexInfo) bool {
	return a.AccessMethod == b.AccessMethod && a.Partial == b.Partial && a.Predicate == b.Predicate
}

// columnsCovered reports whether coveringColumns' leading columns are
// exactly candidateColumns, in order — i.e. every query servable by an
// index on candidateColumns is also servable by an index on
// coveringColumns.
func columnsCovered(coveringColumns, candidateColumns []string) bool {
	if len(coveringColumns) < len(candidateColumns) {
		return false
	}
	for i, col := range candidateColumns {
		if coveringColumns[i] != col {
			return false
		}
	}
	return true
}

// preferOther decides whether "other" should be considered the keeper
// over "candidate" when both cover exactly the same columns — otherwise
// two identical indexes would each flag the other as redundant. A
// primary/unique index is always preferred; failing that, the one whose
// name sorts first is kept, so the decision is deterministic.
func preferOther(candidate, other IndexInfo) bool {
	if len(other.Columns) != len(candidate.Columns) {
		return true // other strictly covers more columns
	}
	if other.Primary || other.Unique {
		return true
	}
	if candidate.Primary || candidate.Unique {
		return false
	}
	return candidate.Name > other.Name
}

func buildDuplicateExplanation(candidate, covering IndexInfo) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "Index %q on (%s) appears redundant — index %q on (%s) already covers the same lookups",
		candidate.Name, strings.Join(candidate.Columns, ", "), covering.Name, strings.Join(covering.Columns, ", "))
	if covering.Primary {
		b.WriteString(" and backs the primary key")
	} else if covering.Unique {
		b.WriteString(" and enforces a unique constraint")
	}
	b.WriteString(". Verify against your actual query patterns before dropping it.")
	return b.String()
}
