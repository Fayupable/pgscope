package domain

import (
	"strings"
	"testing"
	"time"
)

func TestDetectVacuumHealthWarnings(t *testing.T) {
	tests := []struct {
		name  string
		stats []TableVacuumStats
		want  []string // table names expected in the result, in order
	}{
		{
			name: "table below the minimum live-tuple floor is ignored even with a high ratio",
			stats: []TableVacuumStats{
				{Table: "tiny_table", LiveTuples: 10, DeadTuples: 900},
			},
			want: nil,
		},
		{
			name: "table at exactly the minimum live-tuple floor qualifies",
			stats: []TableVacuumStats{
				{Table: "at_floor", LiveTuples: MinLiveTuplesForVacuumCheck, DeadTuples: 10000},
			},
			want: []string{"at_floor"},
		},
		{
			name: "ratio below the warning threshold is ignored",
			stats: []TableVacuumStats{
				{Table: "healthy_table", LiveTuples: 10000, DeadTuples: 100},
			},
			want: nil,
		},
		{
			name: "ratio exactly at the warning threshold qualifies",
			stats: []TableVacuumStats{
				{Table: "at_threshold", LiveTuples: 8000, DeadTuples: 2000}, // 20%
			},
			want: []string{"at_threshold"},
		},
		{
			name: "ratio above the warning threshold qualifies",
			stats: []TableVacuumStats{
				{Table: "bloated_table", LiveTuples: 6000, DeadTuples: 4000}, // 40%
			},
			want: []string{"bloated_table"},
		},
		{
			name: "table with zero rows is ignored, not treated as 100% dead",
			stats: []TableVacuumStats{
				{Table: "empty_table", LiveTuples: 0, DeadTuples: 0},
			},
			want: nil,
		},
		{
			name: "mixed input returns only the qualifying tables, preserving input order",
			stats: []TableVacuumStats{
				{Table: "fine", LiveTuples: 10000, DeadTuples: 100},
				{Table: "bloated_one", LiveTuples: 6000, DeadTuples: 4000},
				{Table: "too_small", LiveTuples: 50, DeadTuples: 900},
				{Table: "bloated_two", LiveTuples: 5000, DeadTuples: 5000},
			},
			want: []string{"bloated_one", "bloated_two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectVacuumHealthWarnings(tt.stats)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectVacuumHealthWarnings() returned %d warnings, want %d (%v)", len(got), len(tt.want), got)
			}
			for i, w := range got {
				if w.Table != tt.want[i] {
					t.Errorf("warning[%d].Table = %q, want %q", i, w.Table, tt.want[i])
				}
			}
		})
	}
}

func TestDetectVacuumHealthWarnings_NeverReturnsNilSlice(t *testing.T) {
	got := DetectVacuumHealthWarnings(nil)
	if got == nil {
		t.Fatal("DetectVacuumHealthWarnings(nil) returned a nil slice, want an empty non-nil slice (callers may rely on this for JSON serialization as [])")
	}
	if len(got) != 0 {
		t.Fatalf("DetectVacuumHealthWarnings(nil) = %v, want empty", got)
	}
}

func TestDetectVacuumHealthWarnings_Explanation(t *testing.T) {
	t.Run("mentions it was never autovacuumed when LastAutovacuum is nil", func(t *testing.T) {
		got := DetectVacuumHealthWarnings([]TableVacuumStats{
			{Table: "never_vacuumed", LiveTuples: 6000, DeadTuples: 4000},
		})

		if len(got) != 1 {
			t.Fatalf("expected exactly 1 warning, got %d", len(got))
		}
		if !strings.Contains(got[0].Explanation, "never been autovacuumed") {
			t.Errorf("Explanation = %q, want it to mention it was never autovacuumed", got[0].Explanation)
		}
		if got[0].LastAutovacuum != nil {
			t.Errorf("LastAutovacuum = %v, want nil", got[0].LastAutovacuum)
		}
	})

	t.Run("mentions the last autovacuum timestamp when present", func(t *testing.T) {
		lastRun := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		got := DetectVacuumHealthWarnings([]TableVacuumStats{
			{Table: "recently_vacuumed", LiveTuples: 6000, DeadTuples: 4000, LastAutovacuum: &lastRun},
		})

		if len(got) != 1 {
			t.Fatalf("expected exactly 1 warning, got %d", len(got))
		}
		if !strings.Contains(got[0].Explanation, "Last autovacuumed at") {
			t.Errorf("Explanation = %q, want it to mention the last autovacuum time", got[0].Explanation)
		}
		if got[0].LastAutovacuum == nil || !got[0].LastAutovacuum.Equal(lastRun) {
			t.Errorf("LastAutovacuum = %v, want %v", got[0].LastAutovacuum, lastRun)
		}
	})
}

func TestTableVacuumStats_DeadTupleRatio(t *testing.T) {
	tests := []struct {
		name  string
		stats TableVacuumStats
		want  float64
	}{
		{
			name: "no rows at all yields zero, not division by zero",
			stats: TableVacuumStats{
				LiveTuples: 0,
				DeadTuples: 0,
			},
			want: 0,
		},
		{
			name: "all live rows yields zero percent dead",
			stats: TableVacuumStats{
				LiveTuples: 1000,
				DeadTuples: 0,
			},
			want: 0,
		},
		{
			name: "all dead rows yields one hundred percent",
			stats: TableVacuumStats{
				LiveTuples: 0,
				DeadTuples: 1000,
			},
			want: 100,
		},
		{
			name: "even split yields fifty percent",
			stats: TableVacuumStats{
				LiveTuples: 500,
				DeadTuples: 500,
			},
			want: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stats.deadTupleRatio(); got != tt.want {
				t.Errorf("deadTupleRatio() = %v, want %v", got, tt.want)
			}
		})
	}
}
