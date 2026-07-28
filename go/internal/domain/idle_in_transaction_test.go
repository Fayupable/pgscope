package domain

import "testing"

func TestDetectIdleInTransactionWarnings(t *testing.T) {
	tests := []struct {
		name     string
		sessions []IdleInTransactionSession
		want     []int32 // PIDs expected in the result, in order
	}{
		{
			name: "below the warning threshold is ignored",
			sessions: []IdleInTransactionSession{
				{PID: 1, User: "app", ApplicationName: "web", IdleSeconds: 59},
			},
			want: nil,
		},
		{
			name: "exactly at the warning threshold qualifies",
			sessions: []IdleInTransactionSession{
				{PID: 2, User: "app", ApplicationName: "web", IdleSeconds: IdleInTransactionWarningSeconds},
			},
			want: []int32{2},
		},
		{
			name: "above the warning threshold qualifies",
			sessions: []IdleInTransactionSession{
				{PID: 3, User: "app", ApplicationName: "web", IdleSeconds: 300},
			},
			want: []int32{3},
		},
		{
			name: "zero idle seconds is ignored",
			sessions: []IdleInTransactionSession{
				{PID: 4, User: "app", ApplicationName: "web", IdleSeconds: 0},
			},
			want: nil,
		},
		{
			name: "mixed input returns only the qualifying sessions, preserving input order",
			sessions: []IdleInTransactionSession{
				{PID: 10, User: "app", ApplicationName: "web", IdleSeconds: 5},
				{PID: 11, User: "app", ApplicationName: "web", IdleSeconds: 120},
				{PID: 12, User: "app", ApplicationName: "web", IdleSeconds: 30},
				{PID: 13, User: "app", ApplicationName: "web", IdleSeconds: 600},
			},
			want: []int32{11, 13},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectIdleInTransactionWarnings(tt.sessions)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectIdleInTransactionWarnings() returned %d warnings, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, pid := range tt.want {
				if got[i].PID != pid {
					t.Errorf("warning[%d].PID = %d, want %d", i, got[i].PID, pid)
				}
			}
		})
	}
}

func TestDetectIdleInTransactionWarnings_NeverReturnsNilSlice(t *testing.T) {
	got := DetectIdleInTransactionWarnings(nil)
	if got == nil {
		t.Fatal("DetectIdleInTransactionWarnings(nil) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectIdleInTransactionWarnings(nil) = %v, want empty", got)
	}
}

func TestDetectIdleInTransactionWarnings_Explanation(t *testing.T) {
	got := DetectIdleInTransactionWarnings([]IdleInTransactionSession{
		{PID: 42, User: "worker", ApplicationName: "batch-job", IdleSeconds: 180},
	})

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 warning, got %d", len(got))
	}
	for _, want := range []string{"42", "worker", "batch-job", "idle_in_transaction_session_timeout"} {
		if !contains(got[0].Explanation, want) {
			t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
		}
	}
}
