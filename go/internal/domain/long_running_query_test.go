package domain

import "testing"

func TestDetectLongRunningQueryWarnings(t *testing.T) {
	tests := []struct {
		name     string
		sessions []LongRunningQuerySession
		want     []int32 // PIDs expected in the result, in order
	}{
		{
			name: "below the warning threshold is ignored",
			sessions: []LongRunningQuerySession{
				{PID: 1, User: "app", ApplicationName: "web", Query: "SELECT 1", RunningSeconds: 10},
			},
			want: nil,
		},
		{
			name: "exactly at the warning threshold qualifies",
			sessions: []LongRunningQuerySession{
				{PID: 2, User: "app", ApplicationName: "web", Query: "SELECT 1", RunningSeconds: LongRunningQueryWarningSeconds},
			},
			want: []int32{2},
		},
		{
			name: "above the warning threshold qualifies",
			sessions: []LongRunningQuerySession{
				{PID: 3, User: "app", ApplicationName: "web", Query: "SELECT 1", RunningSeconds: 900},
			},
			want: []int32{3},
		},
		{
			name: "zero running seconds is ignored",
			sessions: []LongRunningQuerySession{
				{PID: 4, User: "app", ApplicationName: "web", Query: "SELECT 1", RunningSeconds: 0},
			},
			want: nil,
		},
		{
			name: "mixed input returns only the qualifying sessions, preserving input order",
			sessions: []LongRunningQuerySession{
				{PID: 10, User: "app", ApplicationName: "web", Query: "SELECT 1", RunningSeconds: 1},
				{PID: 11, User: "app", ApplicationName: "web", Query: "SELECT 2", RunningSeconds: 600},
				{PID: 12, User: "app", ApplicationName: "web", Query: "SELECT 3", RunningSeconds: 30},
				{PID: 13, User: "app", ApplicationName: "web", Query: "SELECT 4", RunningSeconds: 1200},
			},
			want: []int32{11, 13},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLongRunningQueryWarnings(tt.sessions)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectLongRunningQueryWarnings() returned %d warnings, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, pid := range tt.want {
				if got[i].PID != pid {
					t.Errorf("warning[%d].PID = %d, want %d", i, got[i].PID, pid)
				}
			}
		})
	}
}

func TestDetectLongRunningQueryWarnings_NeverReturnsNilSlice(t *testing.T) {
	got := DetectLongRunningQueryWarnings(nil)
	if got == nil {
		t.Fatal("DetectLongRunningQueryWarnings(nil) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectLongRunningQueryWarnings(nil) = %v, want empty", got)
	}
}

func TestDetectLongRunningQueryWarnings_Explanation(t *testing.T) {
	got := DetectLongRunningQueryWarnings([]LongRunningQuerySession{
		{PID: 42, User: "reporting", ApplicationName: "nightly-job", Query: "SELECT * FROM orders", RunningSeconds: 620},
	})

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 warning, got %d", len(got))
	}
	for _, want := range []string{"42", "reporting", "nightly-job", "620", "normalized", "pg_stat_activity WHERE pid = 42"} {
		if !contains(got[0].Explanation, want) {
			t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
		}
	}
}
