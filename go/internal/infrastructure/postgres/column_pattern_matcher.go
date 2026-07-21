package postgres

import (
	"sort"
	"strings"
)

const suspectedColumnLimit = 2

// extractSuspectedColumns tallies which column names appear most often in
// a "column = $N"-shaped comparison across query texts already known to
// reference the candidate table. Best-effort and advisory — pattern
// matching over already-masked query text, never literal values. Pure
// text processing, no I/O.
func extractSuspectedColumns(queryTexts []string) []string {
	columnCounts := make(map[string]int)

	for _, text := range queryTexts {
		for _, column := range findComparedColumns(text) {
			columnCounts[column]++
		}
	}

	return topColumns(columnCounts, suspectedColumnLimit)
}

func findComparedColumns(text string) []string {
	comparisonOps := []string{"=", ">", "<", ">=", "<="}
	fields := strings.Fields(text)
	columns := make([]string, 0)

	for i := 0; i+2 < len(fields); i++ {
		if !isComparisonOp(fields[i+1], comparisonOps) {
			continue
		}
		if !strings.HasPrefix(fields[i+2], "$") {
			continue
		}
		columns = append(columns, strings.ToLower(fields[i]))
	}

	return columns
}

func isComparisonOp(token string, ops []string) bool {
	for _, op := range ops {
		if token == op {
			return true
		}
	}
	return false
}

func topColumns(counts map[string]int, limit int) []string {
	type entry struct {
		column string
		count  int
	}

	entries := make([]entry, 0, len(counts))
	for column, count := range counts {
		entries = append(entries, entry{column, count})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].column < entries[j].column
	})

	result := make([]string, 0, limit)
	for i := 0; i < len(entries) && i < limit; i++ {
		result = append(result, entries[i].column)
	}

	return result
}
