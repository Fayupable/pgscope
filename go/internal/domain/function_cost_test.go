package domain

import "testing"

func TestDetectExpensiveFunctions(t *testing.T) {
	tests := []struct {
		name  string
		stats []FunctionCallStats
		want  []string // function names expected in the result, in order
	}{
		{
			name: "below the minimum call count is ignored even with high self-time",
			stats: []FunctionCallStats{
				{Function: "rare_fn", Calls: 10, SelfTimeMs: 1000}, // 100ms/call
			},
			want: nil,
		},
		{
			name: "at exactly the minimum call count qualifies",
			stats: []FunctionCallStats{
				{Function: "at_floor_fn", Calls: MinCallsForFunctionCostWarning, SelfTimeMs: 1000}, // 10ms/call
			},
			want: []string{"at_floor_fn"},
		},
		{
			name: "below the minimum self-time per call is ignored",
			stats: []FunctionCallStats{
				{Function: "fast_fn", Calls: 1000, SelfTimeMs: 1000}, // 1ms/call
			},
			want: nil,
		},
		{
			name: "at exactly the minimum self-time per call qualifies",
			stats: []FunctionCallStats{
				{Function: "at_threshold_fn", Calls: 1000, SelfTimeMs: 5000}, // exactly 5ms/call
			},
			want: []string{"at_threshold_fn"},
		},
		{
			name: "above the minimum self-time per call qualifies",
			stats: []FunctionCallStats{
				{Function: "slow_fn", Calls: 1000, SelfTimeMs: 20000}, // 20ms/call
			},
			want: []string{"slow_fn"},
		},
		{
			name: "a function with zero calls is ignored, not treated as infinite self-time",
			stats: []FunctionCallStats{
				{Function: "never_called_fn", Calls: 0, SelfTimeMs: 0},
			},
			want: nil,
		},
		{
			name: "mixed input returns only the qualifying functions, preserving input order",
			stats: []FunctionCallStats{
				{Function: "fine_fn", Calls: 1000, SelfTimeMs: 500},    // 0.5ms/call
				{Function: "slow_one", Calls: 1000, SelfTimeMs: 20000}, // 20ms/call
				{Function: "too_rare", Calls: 5, SelfTimeMs: 1000},     // qualifies on time, not calls
				{Function: "slow_two", Calls: 2000, SelfTimeMs: 30000}, // 15ms/call
			},
			want: []string{"slow_one", "slow_two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectExpensiveFunctions(tt.stats)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectExpensiveFunctions() returned %d results, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, name := range tt.want {
				if got[i].Function != name {
					t.Errorf("result[%d].Function = %q, want %q", i, got[i].Function, name)
				}
			}
		})
	}
}

func TestDetectExpensiveFunctions_NeverReturnsNilSlice(t *testing.T) {
	got := DetectExpensiveFunctions(nil)
	if got == nil {
		t.Fatal("DetectExpensiveFunctions(nil) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectExpensiveFunctions(nil) = %v, want empty", got)
	}
}

func TestDetectExpensiveFunctions_Explanation(t *testing.T) {
	t.Run("mentions trigger tables and the shared-average caveat when IsTrigger is true", func(t *testing.T) {
		got := DetectExpensiveFunctions([]FunctionCallStats{
			{
				Function:      "audit_log_fn",
				Calls:         1000,
				SelfTimeMs:    20000,
				IsTrigger:     true,
				TriggerTables: []string{"orders", "payments"},
			},
		})

		if len(got) != 1 {
			t.Fatalf("expected exactly 1 result, got %d", len(got))
		}
		for _, want := range []string{"used as a trigger", "orders, payments", "can't separate the two"} {
			if !contains(got[0].Explanation, want) {
				t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
			}
		}
	})

	t.Run("does not mention triggers when IsTrigger is false", func(t *testing.T) {
		got := DetectExpensiveFunctions([]FunctionCallStats{
			{Function: "plain_fn", Calls: 1000, SelfTimeMs: 20000},
		})

		if len(got) != 1 {
			t.Fatalf("expected exactly 1 result, got %d", len(got))
		}
		if contains(got[0].Explanation, "trigger") {
			t.Errorf("Explanation = %q, want it to not mention triggers", got[0].Explanation)
		}
	})
}

func TestFunctionCallStats_SelfTimePerCall(t *testing.T) {
	tests := []struct {
		name  string
		stats FunctionCallStats
		want  float64
	}{
		{
			name: "zero calls yields zero, not division by zero",
			stats: FunctionCallStats{
				Calls:      0,
				SelfTimeMs: 500,
			},
			want: 0,
		},
		{
			name: "even split across calls",
			stats: FunctionCallStats{
				Calls:      100,
				SelfTimeMs: 1000,
			},
			want: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stats.selfTimePerCall(); got != tt.want {
				t.Errorf("selfTimePerCall() = %v, want %v", got, tt.want)
			}
		})
	}
}
