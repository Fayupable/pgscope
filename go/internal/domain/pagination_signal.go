package domain

import (
	"fmt"
	"strings"
)

const (
	MinCallsForPaginationWarning  = 100
	MinMeanMsForPaginationWarning = 5.0
	MaxAvgRowsForPagination       = 200.0
	MinVarianceCoefficient        = 1.0
)

// PaginationSignal judges whether a query shape's OFFSET usage looks like
// it may be paginating deep enough to matter. This is inherently a
// heuristic: pgscope only ever sees normalized query text and aggregate
// statistics, never the actual OFFSET values used per call, so the
// verdict is always presented as an inference, never a certainty.
type PaginationSignal struct {
	Query          string
	ContainsOffset bool
	Calls          int64
	MeanExecMs     float64
	StddevExecMs   float64
	Rows           int64
}

func (s PaginationSignal) averageRows() float64 {
	if s.Calls == 0 {
		return 0
	}
	return float64(s.Rows) / float64(s.Calls)
}

func (s PaginationSignal) varianceCoefficient() float64 {
	if s.MeanExecMs <= 0 {
		return 0
	}
	return s.StddevExecMs / s.MeanExecMs
}

// Suspected reports whether this query shape is worth flagging as a
// possible deep-OFFSET pagination pattern.
func (s PaginationSignal) Suspected() bool {
	return s.ContainsOffset &&
		s.Calls >= MinCallsForPaginationWarning &&
		s.MeanExecMs >= MinMeanMsForPaginationWarning &&
		s.averageRows() <= MaxAvgRowsForPagination &&
		s.varianceCoefficient() > MinVarianceCoefficient
}

// Warning returns an empty string when the pattern isn't suspected —
// callers should treat that as "no warning," never render a blank one.
func (s PaginationSignal) Warning() string {
	if !s.Suspected() {
		return ""
	}

	var b strings.Builder
	_, _ = fmt.Fprintf(&b,
		"This query shape uses OFFSET and shows execution-time variance consistent with deep pagination (%d calls, %.2f ms mean, %.2fx variance), averaging %.0f rows per call.",
		s.Calls, s.MeanExecMs, s.varianceCoefficient(), s.averageRows())
	b.WriteString(" This is an inference, not a certainty — pgscope only reads aggregate statistics and cannot see the actual OFFSET values used. Verify against your own data before changing anything.")
	return b.String()
}

// PaginationWarning pairs a flagged query with its explanation — the
// output shape for the dedicated advisory list, independent of whether
// that query shape happens to rank among the costliest overall.
type PaginationWarning struct {
	Query   string `json:"query"`
	Warning string `json:"warning"`
}

// DetectPaginationWarnings filters a set of candidate query signals down
// to the ones actually worth surfacing.
func DetectPaginationWarnings(signals []PaginationSignal) []PaginationWarning {
	result := make([]PaginationWarning, 0)
	for _, s := range signals {
		if !s.Suspected() {
			continue
		}
		result = append(result, PaginationWarning{
			Query:   s.Query,
			Warning: s.Warning(),
		})
	}
	return result
}
