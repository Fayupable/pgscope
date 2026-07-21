package domain

import "math"

const (
	MinRowsForSuggestion  = 1000
	MinTotalScansStrong   = 1000
	MinTotalScansWeak     = 100
	MaxIndexUsagePercent  = 95.0
	StrongSignalThreshold = 10.0
	PoorSelectivityRatio  = 0.05
)

type Confidence string

const (
	ConfidenceStrong       Confidence = "strong"
	ConfidenceWeak         Confidence = "weak"
	ConfidenceInsufficient Confidence = "insufficient_data"
)

// IndexSignal is a pure, deterministic policy for judging whether a
// table's scan statistics justify an index suggestion. It knows nothing
// about how the numbers were fetched — only the numbers themselves —
// so it stays testable without a database.
type IndexSignal struct {
	EstimatedRows int64
	SeqScan       int64
	IdxScan       int64
	// NDistinct is pg_stats.n_distinct for the suspected column, if one
	// was found and stats were available. Nil means "unknown" — the
	// signal falls back to scan-based confidence only.
	NDistinct *float64
}

func (s IndexSignal) TotalScans() int64 {
	return s.SeqScan + s.IdxScan
}

func (s IndexSignal) IndexUsagePercent() float64 {
	if s.TotalScans() == 0 {
		return 100
	}
	return 100 * float64(s.IdxScan) / float64(s.TotalScans())
}

func (s IndexSignal) SeqScanRatio() float64 {
	if s.TotalScans() == 0 {
		return 0
	}
	return float64(s.SeqScan) / float64(s.TotalScans())
}

// Strength rewards volume, not just ratio, so a lopsided ratio on a
// handful of scans can't outrank a real, high-volume pattern.
func (s IndexSignal) Strength() float64 {
	return math.Log2(1+float64(s.SeqScan)) * s.SeqScanRatio()
}

// Selectivity estimates the fraction of rows a typical equality lookup
// on the suspected column would return, derived from pg_stats.n_distinct.
// Lower is better (more selective). Returns false if unknown.
func (s IndexSignal) Selectivity() (float64, bool) {
	if s.NDistinct == nil || s.EstimatedRows <= 0 {
		return 0, false
	}

	n := *s.NDistinct
	var distinctCount float64
	switch {
	case n > 0:
		distinctCount = n
	case n < 0:
		distinctCount = -n * float64(s.EstimatedRows)
	default:
		return 0, false
	}

	if distinctCount <= 0 {
		return 0, false
	}

	return 1.0 / distinctCount, true
}

// Qualifies is the minimum bar before this table is considered at all —
// mirrors pgHero's proven production thresholds (n_live_tup, idx_scan>0,
// index-usage percentage), not a value invented for this project.
func (s IndexSignal) Qualifies() bool {
	return s.EstimatedRows >= MinRowsForSuggestion &&
		s.IdxScan > 0 &&
		s.IndexUsagePercent() < MaxIndexUsagePercent
}

func (s IndexSignal) Confidence() Confidence {
	if !s.Qualifies() {
		return ConfidenceInsufficient
	}

	confidence := ConfidenceInsufficient
	if s.TotalScans() >= MinTotalScansStrong && s.Strength() >= StrongSignalThreshold {
		confidence = ConfidenceStrong
	} else if s.TotalScans() >= MinTotalScansWeak {
		confidence = ConfidenceWeak
	}

	// A low-cardinality column rarely benefits from an index even under
	// heavy scan pressure — the planner will keep choosing a seq scan.
	// Demote rather than suppress: the scan pattern is still real signal.
	if selectivity, known := s.Selectivity(); known && selectivity > PoorSelectivityRatio && confidence == ConfidenceStrong {
		confidence = ConfidenceWeak
	}

	return confidence
}
