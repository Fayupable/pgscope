package domain

import "testing"

func TestDetectReplicationLagWarnings(t *testing.T) {
	tests := []struct {
		name     string
		replicas []ReplicaLagInfo
		want     []string // application names expected in the result, in order
	}{
		{
			name: "below the warning threshold is ignored",
			replicas: []ReplicaLagInfo{
				{ApplicationName: "replica1", LagBytes: ReplicationLagWarningBytes - 1},
			},
			want: nil,
		},
		{
			name: "exactly at the warning threshold qualifies",
			replicas: []ReplicaLagInfo{
				{ApplicationName: "replica2", LagBytes: ReplicationLagWarningBytes},
			},
			want: []string{"replica2"},
		},
		{
			name: "above the warning threshold qualifies",
			replicas: []ReplicaLagInfo{
				{ApplicationName: "replica3", LagBytes: 500 * 1024 * 1024},
			},
			want: []string{"replica3"},
		},
		{
			name: "zero lag is ignored",
			replicas: []ReplicaLagInfo{
				{ApplicationName: "replica4", LagBytes: 0},
			},
			want: nil,
		},
		{
			name: "mixed input returns only the qualifying replicas, preserving input order",
			replicas: []ReplicaLagInfo{
				{ApplicationName: "in_sync", LagBytes: 1024},
				{ApplicationName: "lagging_one", LagBytes: 200 * 1024 * 1024},
				{ApplicationName: "almost_synced", LagBytes: 1024 * 1024},
				{ApplicationName: "lagging_two", LagBytes: 300 * 1024 * 1024},
			},
			want: []string{"lagging_one", "lagging_two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectReplicationLagWarnings(tt.replicas)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectReplicationLagWarnings() returned %d warnings, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, name := range tt.want {
				if got[i].ApplicationName != name {
					t.Errorf("warning[%d].ApplicationName = %q, want %q", i, got[i].ApplicationName, name)
				}
			}
		})
	}
}

func TestDetectReplicationLagWarnings_NeverReturnsNilSlice(t *testing.T) {
	got := DetectReplicationLagWarnings(nil)
	if got == nil {
		t.Fatal("DetectReplicationLagWarnings(nil) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectReplicationLagWarnings(nil) = %v, want empty", got)
	}
}

func TestDetectReplicationLagWarnings_Explanation(t *testing.T) {
	got := DetectReplicationLagWarnings([]ReplicaLagInfo{
		{ApplicationName: "replica_west", ClientAddr: "10.0.0.5", LagBytes: 250 * 1024 * 1024},
	})

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 warning, got %d", len(got))
	}
	for _, want := range []string{"replica_west", "10.0.0.5", "250 MB"} {
		if !contains(got[0].Explanation, want) {
			t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
		}
	}
}
