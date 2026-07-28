package domain

import "testing"

func TestDetectReplicationSlotWarnings(t *testing.T) {
	tests := []struct {
		name  string
		slots []ReplicationSlotInfo
		want  []string // slot names expected in the result, in order
	}{
		{
			name: "below the warning threshold is ignored",
			slots: []ReplicationSlotInfo{
				{SlotName: "slot_a", Active: true, RetainedBytes: ReplicationSlotWarningBytes - 1},
			},
			want: nil,
		},
		{
			name: "exactly at the warning threshold qualifies",
			slots: []ReplicationSlotInfo{
				{SlotName: "slot_b", Active: true, RetainedBytes: ReplicationSlotWarningBytes},
			},
			want: []string{"slot_b"},
		},
		{
			name: "above the warning threshold qualifies",
			slots: []ReplicationSlotInfo{
				{SlotName: "slot_c", Active: true, RetainedBytes: 2 * ReplicationSlotWarningBytes},
			},
			want: []string{"slot_c"},
		},
		{
			name: "an inactive slot retaining a lot of WAL also qualifies",
			slots: []ReplicationSlotInfo{
				{SlotName: "slot_orphaned", Active: false, RetainedBytes: 3 * ReplicationSlotWarningBytes},
			},
			want: []string{"slot_orphaned"},
		},
		{
			name: "zero retained bytes is ignored",
			slots: []ReplicationSlotInfo{
				{SlotName: "slot_fresh", Active: true, RetainedBytes: 0},
			},
			want: nil,
		},
		{
			name: "mixed input returns only the qualifying slots, preserving input order",
			slots: []ReplicationSlotInfo{
				{SlotName: "healthy_slot", Active: true, RetainedBytes: 1024},
				{SlotName: "bloated_one", Active: true, RetainedBytes: 2 * ReplicationSlotWarningBytes},
				{SlotName: "also_healthy", Active: true, RetainedBytes: 1024 * 1024},
				{SlotName: "bloated_two", Active: false, RetainedBytes: 5 * ReplicationSlotWarningBytes},
			},
			want: []string{"bloated_one", "bloated_two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectReplicationSlotWarnings(tt.slots)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectReplicationSlotWarnings() returned %d warnings, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, name := range tt.want {
				if got[i].SlotName != name {
					t.Errorf("warning[%d].SlotName = %q, want %q", i, got[i].SlotName, name)
				}
			}
		})
	}
}

func TestDetectReplicationSlotWarnings_NeverReturnsNilSlice(t *testing.T) {
	got := DetectReplicationSlotWarnings(nil)
	if got == nil {
		t.Fatal("DetectReplicationSlotWarnings(nil) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectReplicationSlotWarnings(nil) = %v, want empty", got)
	}
}

func TestDetectReplicationSlotWarnings_Explanation(t *testing.T) {
	t.Run("mentions the slot may be orphaned when inactive", func(t *testing.T) {
		got := DetectReplicationSlotWarnings([]ReplicationSlotInfo{
			{SlotName: "old_replica_slot", Active: false, RetainedBytes: 2 * ReplicationSlotWarningBytes},
		})

		if len(got) != 1 {
			t.Fatalf("expected exactly 1 warning, got %d", len(got))
		}
		for _, want := range []string{"old_replica_slot", "inactive", "before dropping it"} {
			if !contains(got[0].Explanation, want) {
				t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
			}
		}
	})

	t.Run("mentions checking connectivity when active", func(t *testing.T) {
		got := DetectReplicationSlotWarnings([]ReplicationSlotInfo{
			{SlotName: "live_replica_slot", Active: true, RetainedBytes: 2 * ReplicationSlotWarningBytes},
		})

		if len(got) != 1 {
			t.Fatalf("expected exactly 1 warning, got %d", len(got))
		}
		for _, want := range []string{"live_replica_slot", "connectivity and load"} {
			if !contains(got[0].Explanation, want) {
				t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
			}
		}
	})
}
