package domain

import "testing"

func TestDetectPhysicalIOHotspots(t *testing.T) {
	tests := []struct {
		name  string
		stats []QueryPhysicalIO
		want  []string // queries expected in the result, in order
	}{
		{
			name: "below the minimum exec reads is ignored",
			stats: []QueryPhysicalIO{
				{Query: "cached_query", ExecReads: 999},
			},
			want: nil,
		},
		{
			name: "exactly at the minimum exec reads qualifies",
			stats: []QueryPhysicalIO{
				{Query: "at_floor_query", ExecReads: MinExecReadsForKcacheWarning},
			},
			want: []string{"at_floor_query"},
		},
		{
			name: "above the minimum exec reads qualifies",
			stats: []QueryPhysicalIO{
				{Query: "hotspot_query", ExecReads: 50000},
			},
			want: []string{"hotspot_query"},
		},
		{
			name: "zero exec reads is ignored",
			stats: []QueryPhysicalIO{
				{Query: "no_reads_query", ExecReads: 0},
			},
			want: nil,
		},
		{
			name: "mixed input returns only the qualifying queries, preserving input order",
			stats: []QueryPhysicalIO{
				{Query: "fine_query", ExecReads: 100},
				{Query: "hot_one", ExecReads: 5000},
				{Query: "also_fine", ExecReads: 500},
				{Query: "hot_two", ExecReads: 10000},
			},
			want: []string{"hot_one", "hot_two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectPhysicalIOHotspots(tt.stats)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectPhysicalIOHotspots() returned %d results, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, q := range tt.want {
				if got[i].Query != q {
					t.Errorf("result[%d].Query = %q, want %q", i, got[i].Query, q)
				}
			}
		})
	}
}

func TestDetectPhysicalIOHotspots_NeverReturnsNilSlice(t *testing.T) {
	got := DetectPhysicalIOHotspots(nil)
	if got == nil {
		t.Fatal("DetectPhysicalIOHotspots(nil) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectPhysicalIOHotspots(nil) = %v, want empty", got)
	}
}

func TestDetectPhysicalIOHotspots_Explanation(t *testing.T) {
	got := DetectPhysicalIOHotspots([]QueryPhysicalIO{
		{
			Query:        "select * from big_table",
			Calls:        42,
			ExecReads:    5000,
			ExecWrites:   10,
			UserTimeMs:   12.5,
			SystemTimeMs: 3.2,
		},
	})

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 result, got %d", len(got))
	}
	for _, want := range []string{"5000", "10", "42", "shared_buffers"} {
		if !contains(got[0].Explanation, want) {
			t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
		}
	}
}
