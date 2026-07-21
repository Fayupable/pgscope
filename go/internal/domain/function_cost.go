package domain

import (
	"fmt"
	"strings"
)

const (
	MinCallsForFunctionCostWarning = 100
	MinSelfTimePerCallMs           = 5.0
)

// FunctionCallStats is the raw shape read from pg_stat_user_functions,
// joined with pg_trigger to know whether the function is used as a
// trigger and on which tables.
type FunctionCallStats struct {
	Function      string
	Calls         int64
	TotalTimeMs   float64
	SelfTimeMs    float64
	IsTrigger     bool
	TriggerTables []string
}

func (s FunctionCallStats) selfTimePerCall() float64 {
	if s.Calls == 0 {
		return 0
	}
	return s.SelfTimeMs / float64(s.Calls)
}

func (s FunctionCallStats) worthFlagging() bool {
	return s.Calls >= MinCallsForFunctionCostWarning && s.selfTimePerCall() >= MinSelfTimePerCallMs
}

// FunctionCost is a suggestion, never a certainty — mirrors the tone of
// every other advisory in this package. A high self-time-per-call
// average may be entirely intentional (audit logging, denormalization),
// so the wording never implies the function is wrong, only that it's
// worth a look.
type FunctionCost struct {
	Function      string   `json:"function"`
	Calls         int64    `json:"calls"`
	SelfTimeMs    float64  `json:"selfTimeMs"`
	IsTrigger     bool     `json:"isTrigger"`
	TriggerTables []string `json:"triggerTables,omitempty"`
	Explanation   string   `json:"explanation"`
}

// DetectExpensiveFunctions filters and explains function/trigger call
// stats worth surfacing. Functions below the minimum call count or
// average self-time are silently skipped — noise, not signal.
func DetectExpensiveFunctions(stats []FunctionCallStats) []FunctionCost {
	result := make([]FunctionCost, 0)
	for _, s := range stats {
		if !s.worthFlagging() {
			continue
		}
		result = append(result, FunctionCost{
			Function:      s.Function,
			Calls:         s.Calls,
			SelfTimeMs:    s.SelfTimeMs,
			IsTrigger:     s.IsTrigger,
			TriggerTables: s.TriggerTables,
			Explanation:   buildFunctionCostExplanation(s),
		})
	}
	return result
}

func buildFunctionCostExplanation(s FunctionCallStats) string {
	perCall := s.selfTimePerCall()
	if s.IsTrigger {
		return fmt.Sprintf(
			"Function %q is used as a trigger on %s and averages %.2f ms of self-time per call over %d calls. This may be expected if the trigger performs essential work (logging, denormalization, validation). If it's contributing to write latency you're trying to reduce, consider optimizing the logic or moving non-critical work to a background job. Note: this average includes any direct (non-trigger) calls to the same function too, since pgscope can't separate the two.",
			s.Function, strings.Join(s.TriggerTables, ", "), perCall, s.Calls,
		)
	}
	return fmt.Sprintf(
		"Function %q averages %.2f ms of self-time per call over %d calls. Worth a look if it's on a latency-sensitive path — this isn't necessarily a problem, just a signal.",
		s.Function, perCall, s.Calls,
	)
}
