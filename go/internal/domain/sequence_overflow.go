package domain

import "fmt"

const SequenceOverflowWarningPercent = 75.0

// SequenceUsage is the raw shape read from the system catalog for one
// non-cycling sequence — no judgment applied yet. Cycling sequences are
// excluded upstream since wrapping around is their intended behavior,
// not a risk.
type SequenceUsage struct {
	Sequence     string
	CurrentValue int64
	MaxValue     int64
}

// SequenceOverflowRisk is a suggestion, never a certainty — a sequence
// nearing its data type's maximum value will eventually cause inserts
// to fail once exhausted. This is usually a slow-moving, easy-to-miss
// risk on old or high-throughput tables.
type SequenceOverflowRisk struct {
	Sequence     string  `json:"sequence"`
	CurrentValue int64   `json:"currentValue"`
	MaxValue     int64   `json:"maxValue"`
	UsagePercent float64 `json:"usagePercent"`
	Explanation  string  `json:"explanation"`
}

// DetectSequenceOverflowRisks filters sequences down to the ones
// meaningfully close to exhausting their range.
func DetectSequenceOverflowRisks(sequences []SequenceUsage) []SequenceOverflowRisk {
	result := make([]SequenceOverflowRisk, 0)
	for _, s := range sequences {
		if s.MaxValue <= 0 {
			continue
		}

		usage := 100 * float64(s.CurrentValue) / float64(s.MaxValue)
		if usage < SequenceOverflowWarningPercent {
			continue
		}

		result = append(result, SequenceOverflowRisk{
			Sequence:     s.Sequence,
			CurrentValue: s.CurrentValue,
			MaxValue:     s.MaxValue,
			UsagePercent: usage,
			Explanation: fmt.Sprintf(
				"Sequence %q is at %.0f%% of its maximum value (%d of %d). If it's exhausted, inserts relying on it will start failing. Consider migrating the underlying column to bigint before that happens.",
				s.Sequence, usage, s.CurrentValue, s.MaxValue,
			),
		})
	}
	return result
}
