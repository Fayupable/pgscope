package domain

import "testing"

func TestDetectSequenceOverflowRisks(t *testing.T) {
	tests := []struct {
		name      string
		sequences []SequenceUsage
		want      []string // sequence names expected in the result, in order
	}{
		{
			name: "max value zero or negative is ignored to avoid division by zero",
			sequences: []SequenceUsage{
				{Sequence: "broken_seq", CurrentValue: 100, MaxValue: 0},
			},
			want: nil,
		},
		{
			name: "below the warning threshold is ignored",
			sequences: []SequenceUsage{
				{Sequence: "healthy_seq", CurrentValue: 50, MaxValue: 100}, // 50%
			},
			want: nil,
		},
		{
			name: "exactly at the warning threshold qualifies",
			sequences: []SequenceUsage{
				{Sequence: "at_threshold_seq", CurrentValue: 75, MaxValue: 100}, // exactly 75%
			},
			want: []string{"at_threshold_seq"},
		},
		{
			name: "above the warning threshold qualifies",
			sequences: []SequenceUsage{
				{Sequence: "near_max_seq", CurrentValue: 95, MaxValue: 100}, // 95%
			},
			want: []string{"near_max_seq"},
		},
		{
			name: "fully exhausted qualifies",
			sequences: []SequenceUsage{
				{Sequence: "exhausted_seq", CurrentValue: 100, MaxValue: 100}, // 100%
			},
			want: []string{"exhausted_seq"},
		},
		{
			name: "mixed input returns only the qualifying sequences, preserving input order",
			sequences: []SequenceUsage{
				{Sequence: "fine_seq", CurrentValue: 10, MaxValue: 100},
				{Sequence: "risky_one", CurrentValue: 80, MaxValue: 100},
				{Sequence: "also_fine", CurrentValue: 50, MaxValue: 100},
				{Sequence: "risky_two", CurrentValue: 90, MaxValue: 100},
			},
			want: []string{"risky_one", "risky_two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectSequenceOverflowRisks(tt.sequences)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectSequenceOverflowRisks() returned %d results, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, name := range tt.want {
				if got[i].Sequence != name {
					t.Errorf("result[%d].Sequence = %q, want %q", i, got[i].Sequence, name)
				}
			}
		})
	}
}

func TestDetectSequenceOverflowRisks_NeverReturnsNilSlice(t *testing.T) {
	got := DetectSequenceOverflowRisks(nil)
	if got == nil {
		t.Fatal("DetectSequenceOverflowRisks(nil) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectSequenceOverflowRisks(nil) = %v, want empty", got)
	}
}

func TestDetectSequenceOverflowRisks_Explanation(t *testing.T) {
	got := DetectSequenceOverflowRisks([]SequenceUsage{
		{Sequence: "orders_id_seq", CurrentValue: 90, MaxValue: 100},
	})

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 result, got %d", len(got))
	}
	for _, want := range []string{"orders_id_seq", "90%", "90 of 100", "bigint"} {
		if !contains(got[0].Explanation, want) {
			t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
		}
	}
}
