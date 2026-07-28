package domain

import "testing"

func TestDetectPreparedTransactionWarnings(t *testing.T) {
	tests := []struct {
		name         string
		transactions []PreparedTransactionInfo
		want         []string // GIDs expected in the result, in order
	}{
		{
			name: "below the warning threshold is ignored",
			transactions: []PreparedTransactionInfo{
				{GID: "tx_recent", Database: "app", Owner: "app_user", AgeSeconds: 5},
			},
			want: nil,
		},
		{
			name: "exactly at the warning threshold qualifies",
			transactions: []PreparedTransactionInfo{
				{GID: "tx_at_threshold", Database: "app", Owner: "app_user", AgeSeconds: PreparedTransactionWarningSeconds},
			},
			want: []string{"tx_at_threshold"},
		},
		{
			name: "above the warning threshold qualifies",
			transactions: []PreparedTransactionInfo{
				{GID: "tx_stale", Database: "app", Owner: "app_user", AgeSeconds: 3600},
			},
			want: []string{"tx_stale"},
		},
		{
			name: "zero age is ignored",
			transactions: []PreparedTransactionInfo{
				{GID: "tx_fresh", Database: "app", Owner: "app_user", AgeSeconds: 0},
			},
			want: nil,
		},
		{
			name: "mixed input returns only the qualifying transactions, preserving input order",
			transactions: []PreparedTransactionInfo{
				{GID: "tx_fine", Database: "app", Owner: "app_user", AgeSeconds: 2},
				{GID: "tx_stale_one", Database: "app", Owner: "app_user", AgeSeconds: 900},
				{GID: "tx_also_fine", Database: "app", Owner: "app_user", AgeSeconds: 30},
				{GID: "tx_stale_two", Database: "app", Owner: "app_user", AgeSeconds: 1800},
			},
			want: []string{"tx_stale_one", "tx_stale_two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectPreparedTransactionWarnings(tt.transactions)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectPreparedTransactionWarnings() returned %d warnings, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, gid := range tt.want {
				if got[i].GID != gid {
					t.Errorf("warning[%d].GID = %q, want %q", i, got[i].GID, gid)
				}
			}
		})
	}
}

func TestDetectPreparedTransactionWarnings_NeverReturnsNilSlice(t *testing.T) {
	got := DetectPreparedTransactionWarnings(nil)
	if got == nil {
		t.Fatal("DetectPreparedTransactionWarnings(nil) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectPreparedTransactionWarnings(nil) = %v, want empty", got)
	}
}

func TestDetectPreparedTransactionWarnings_Explanation(t *testing.T) {
	got := DetectPreparedTransactionWarnings([]PreparedTransactionInfo{
		{GID: "tx_abandoned", Database: "orders_db", Owner: "batch_worker", AgeSeconds: 1200},
	})

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 warning, got %d", len(got))
	}
	for _, want := range []string{"tx_abandoned", "orders_db", "batch_worker", "1200", "COMMIT PREPARED", "ROLLBACK PREPARED"} {
		if !contains(got[0].Explanation, want) {
			t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
		}
	}
}
