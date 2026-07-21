package domain

import (
	"fmt"
	"strings"
)

// NewIndexCandidate builds a fully-judged IndexCandidate from raw
// statistics. All confidence scoring and wording decisions happen here,
// in the domain layer, so infrastructure adapters stay pure I/O and
// never decide what a number "means."
func NewIndexCandidate(table string, signal IndexSignal, suspectedColumns []string, existingIndexCount int, writesPerSecond float64) IndexCandidate {
	confidence := signal.Confidence()
	selectivity, selectivityKnown := signal.Selectivity()

	candidate := IndexCandidate{
		Table:              table,
		SeqScanCount:       signal.SeqScan,
		IdxScanCount:       signal.IdxScan,
		SuspectedColumns:   suspectedColumns,
		ExistingIndexCount: existingIndexCount,
		WritesPerSecond:    writesPerSecond,
		Confidence:         confidence,
		Rationale:          buildRationale(table, signal, suspectedColumns, existingIndexCount, writesPerSecond, confidence),
	}

	if selectivityKnown {
		candidate.Selectivity = &selectivity
	}

	return candidate
}

func buildRationale(table string, signal IndexSignal, suspectedColumns []string, existingIndexCount int, writesPerSecond float64, confidence Confidence) string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "%s: %d sequential scans vs %d index scans (%.0f%% index usage)", table, signal.SeqScan, signal.IdxScan, signal.IndexUsagePercent())

	if len(suspectedColumns) > 0 {
		_, _ = fmt.Fprintf(&b, ", most often filtering on %s", strings.Join(suspectedColumns, ", "))
	}
	b.WriteString(".")

	switch confidence {
	case ConfidenceStrong:
		b.WriteString(" This is a strong, high-volume signal.")
	case ConfidenceWeak:
		b.WriteString(" Signal is present but based on a smaller sample — keep monitoring.")
	default:
		b.WriteString(" Not enough data yet to suggest anything confidently.")
	}

	if selectivity, known := signal.Selectivity(); known {
		if selectivity > PoorSelectivityRatio {
			_, _ = fmt.Fprintf(&b, " The suspected column has low selectivity (an average lookup returns ~%.0f%% of rows), so an index may not help much.", selectivity*100)
		} else {
			_, _ = fmt.Fprintf(&b, " The suspected column looks selective (an average lookup returns ~%.2f%% of rows), which favors an index.", selectivity*100)
		}
	}

	if existingIndexCount == 0 {
		b.WriteString(" This table currently has no indexes at all.")
	}

	if writesPerSecond > 1 {
		_, _ = fmt.Fprintf(&b, " It also receives roughly %.1f writes/s, so a new index would add real overhead — test before applying.", writesPerSecond)
	} else {
		b.WriteString(" Write volume looks low, so a new index is likely low-risk here — still worth testing.")
	}

	return b.String()
}
