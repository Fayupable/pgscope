package domain

import "testing"

func TestDetectLockWaitWarnings(t *testing.T) {
	tests := []struct {
		name     string
		sessions []LockWaitSession
		want     []int32 // WaitingPIDs expected in the result, in order
	}{
		{
			name: "below the warning threshold is ignored",
			sessions: []LockWaitSession{
				{WaitingPID: 1, BlockingPID: 2, LockedTable: "orders", WaitAgeSeconds: 1},
			},
			want: nil,
		},
		{
			name: "exactly at the warning threshold qualifies",
			sessions: []LockWaitSession{
				{WaitingPID: 3, BlockingPID: 4, LockedTable: "orders", WaitAgeSeconds: LockWaitWarningSeconds},
			},
			want: []int32{3},
		},
		{
			name: "above the warning threshold qualifies",
			sessions: []LockWaitSession{
				{WaitingPID: 5, BlockingPID: 6, LockedTable: "orders", WaitAgeSeconds: 60},
			},
			want: []int32{5},
		},
		{
			name: "zero wait age is ignored",
			sessions: []LockWaitSession{
				{WaitingPID: 7, BlockingPID: 8, LockedTable: "orders", WaitAgeSeconds: 0},
			},
			want: nil,
		},
		{
			name: "mixed input returns only the qualifying sessions, preserving input order",
			sessions: []LockWaitSession{
				{WaitingPID: 10, BlockingPID: 20, LockedTable: "a", WaitAgeSeconds: 0.5},
				{WaitingPID: 11, BlockingPID: 21, LockedTable: "b", WaitAgeSeconds: 30},
				{WaitingPID: 12, BlockingPID: 22, LockedTable: "c", WaitAgeSeconds: 2},
				{WaitingPID: 13, BlockingPID: 23, LockedTable: "d", WaitAgeSeconds: 120},
			},
			want: []int32{11, 13},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLockWaitWarnings(tt.sessions)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectLockWaitWarnings() returned %d warnings, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, pid := range tt.want {
				if got[i].WaitingPID != pid {
					t.Errorf("warning[%d].WaitingPID = %d, want %d", i, got[i].WaitingPID, pid)
				}
			}
		})
	}
}

func TestDetectLockWaitWarnings_NeverReturnsNilSlice(t *testing.T) {
	got := DetectLockWaitWarnings(nil)
	if got == nil {
		t.Fatal("DetectLockWaitWarnings(nil) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectLockWaitWarnings(nil) = %v, want empty", got)
	}
}

func TestDetectLockWaitWarnings_Explanation(t *testing.T) {
	got := DetectLockWaitWarnings([]LockWaitSession{
		{
			WaitingPID:     42,
			WaitingQuery:   "UPDATE orders SET status = ? WHERE id = ?",
			BlockingPID:    99,
			BlockingQuery:  "UPDATE orders SET status = ? WHERE id = ?",
			LockedTable:    "orders",
			WaitAgeSeconds: 45,
		},
	})

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 warning, got %d", len(got))
	}
	for _, want := range []string{"42", "99", "orders", "45.0"} {
		if !contains(got[0].Explanation, want) {
			t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
		}
	}
}
