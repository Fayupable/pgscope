package domain

import "testing"

func TestDetectUnusedIndexes(t *testing.T) {
	tests := []struct {
		name            string
		indexes         []UnusedIndexInfo
		statsAgeSeconds float64
		want            []string // index names expected in the result, in order
	}{
		{
			name: "stats age below the minimum observation window returns nothing, regardless of scan count",
			indexes: []UnusedIndexInfo{
				{Table: "orders", Index: "idx_never_used", IndexScans: 0},
			},
			statsAgeSeconds: MinStatsAgeSecondsForUnused - 1,
			want:            nil,
		},
		{
			name: "stats age exactly at the minimum observation window is accepted",
			indexes: []UnusedIndexInfo{
				{Table: "orders", Index: "idx_never_used", IndexScans: 0},
			},
			statsAgeSeconds: MinStatsAgeSecondsForUnused,
			want:            []string{"idx_never_used"},
		},
		{
			name: "index scans above the max-scans floor is not flagged",
			indexes: []UnusedIndexInfo{
				{Table: "orders", Index: "idx_used_often", IndexScans: 51},
			},
			statsAgeSeconds: MinStatsAgeSecondsForUnused,
			want:            nil,
		},
		{
			name: "index scans exactly at the max-scans floor is still flagged",
			indexes: []UnusedIndexInfo{
				{Table: "orders", Index: "idx_at_floor", IndexScans: MaxScansForUnused},
			},
			statsAgeSeconds: MinStatsAgeSecondsForUnused,
			want:            []string{"idx_at_floor"},
		},
		{
			name: "zero scans over a long observation window is flagged",
			indexes: []UnusedIndexInfo{
				{Table: "orders", Index: "idx_dead", IndexScans: 0},
			},
			statsAgeSeconds: MinStatsAgeSecondsForUnused * 10,
			want:            []string{"idx_dead"},
		},
		{
			name: "mixed input returns only the qualifying indexes, preserving input order",
			indexes: []UnusedIndexInfo{
				{Table: "orders", Index: "idx_busy", IndexScans: 10000},
				{Table: "orders", Index: "idx_quiet_one", IndexScans: 2},
				{Table: "orders", Index: "idx_moderate", IndexScans: 500},
				{Table: "orders", Index: "idx_quiet_two", IndexScans: 0},
			},
			statsAgeSeconds: MinStatsAgeSecondsForUnused,
			want:            []string{"idx_quiet_one", "idx_quiet_two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectUnusedIndexes(tt.indexes, tt.statsAgeSeconds)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectUnusedIndexes() returned %d results, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, name := range tt.want {
				if got[i].Index != name {
					t.Errorf("result[%d].Index = %q, want %q", i, got[i].Index, name)
				}
			}
		})
	}
}

func TestDetectUnusedIndexes_NeverReturnsNilSlice(t *testing.T) {
	got := DetectUnusedIndexes(nil, MinStatsAgeSecondsForUnused)
	if got == nil {
		t.Fatal("DetectUnusedIndexes(nil, ...) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectUnusedIndexes(nil, ...) = %v, want empty", got)
	}

	gotTooYoung := DetectUnusedIndexes([]UnusedIndexInfo{{Table: "orders", Index: "idx_x"}}, 0)
	if gotTooYoung == nil {
		t.Fatal("DetectUnusedIndexes(..., 0) returned a nil slice, want an empty non-nil slice")
	}
}

func TestDetectUnusedIndexes_Explanation(t *testing.T) {
	got := DetectUnusedIndexes([]UnusedIndexInfo{
		{Table: "orders", Index: "idx_stale", SizeBytes: 2048, IndexScans: 3},
	}, MinStatsAgeSecondsForUnused*2) // 2 hours

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 result, got %d", len(got))
	}
	for _, want := range []string{"idx_stale", "orders", "3 time(s)", "2.0 hours", "2.0 KiB"} {
		if !contains(got[0].Explanation, want) {
			t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{name: "zero bytes", bytes: 0, want: "0 B"},
		{name: "below one KiB stays in bytes", bytes: 512, want: "512 B"},
		{name: "just below one KiB stays in bytes", bytes: 1023, want: "1023 B"},
		{name: "exactly one KiB", bytes: 1024, want: "1.0 KiB"},
		{name: "one and a half KiB", bytes: 1536, want: "1.5 KiB"},
		{name: "exactly one MiB", bytes: 1024 * 1024, want: "1.0 MiB"},
		{name: "exactly one GiB", bytes: 1024 * 1024 * 1024, want: "1.0 GiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBytes(tt.bytes); got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
