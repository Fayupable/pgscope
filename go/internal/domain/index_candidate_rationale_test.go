package domain

import "testing"

func TestNewIndexCandidate_Fields(t *testing.T) {
	signal := IndexSignal{EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10}

	got := NewIndexCandidate("orders", signal, []string{"user_id"}, 2, 3.5)

	if got.Table != "orders" {
		t.Errorf("Table = %q, want %q", got.Table, "orders")
	}
	if got.SeqScanCount != 5000 {
		t.Errorf("SeqScanCount = %d, want %d", got.SeqScanCount, 5000)
	}
	if got.IdxScanCount != 10 {
		t.Errorf("IdxScanCount = %d, want %d", got.IdxScanCount, 10)
	}
	if len(got.SuspectedColumns) != 1 || got.SuspectedColumns[0] != "user_id" {
		t.Errorf("SuspectedColumns = %v, want [user_id]", got.SuspectedColumns)
	}
	if got.ExistingIndexCount != 2 {
		t.Errorf("ExistingIndexCount = %d, want %d", got.ExistingIndexCount, 2)
	}
	if got.WritesPerSecond != 3.5 {
		t.Errorf("WritesPerSecond = %v, want %v", got.WritesPerSecond, 3.5)
	}
	if got.Confidence != ConfidenceStrong {
		t.Errorf("Confidence = %v, want %v", got.Confidence, ConfidenceStrong)
	}
}

func TestNewIndexCandidate_SelectivityPointer(t *testing.T) {
	t.Run("nil when NDistinct is unknown", func(t *testing.T) {
		signal := IndexSignal{EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10}
		got := NewIndexCandidate("orders", signal, nil, 0, 0)
		if got.Selectivity != nil {
			t.Errorf("Selectivity = %v, want nil", *got.Selectivity)
		}
	})

	t.Run("set when NDistinct is known", func(t *testing.T) {
		ndistinct := 90000.0
		signal := IndexSignal{EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10, NDistinct: &ndistinct}
		got := NewIndexCandidate("orders", signal, nil, 0, 0)
		if got.Selectivity == nil {
			t.Fatal("Selectivity = nil, want a value")
		}
		want := 1.0 / 90000
		if *got.Selectivity < want*0.99 || *got.Selectivity > want*1.01 {
			t.Errorf("Selectivity = %v, want approx %v", *got.Selectivity, want)
		}
	})
}

func TestNewIndexCandidate_Rationale(t *testing.T) {
	tests := []struct {
		name               string
		signal             IndexSignal
		suspectedColumns   []string
		existingIndexCount int
		writesPerSecond    float64
		wantContains       []string
		wantNotContains    []string
	}{
		{
			name:             "strong confidence mentions the strong signal wording",
			signal:           IndexSignal{EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10},
			suspectedColumns: []string{"user_id"},
			wantContains:     []string{"orders", "5000 sequential scans", "10 index scans", "filtering on user_id", "strong, high-volume signal"},
		},
		{
			name:         "weak confidence mentions the smaller-sample wording",
			signal:       IndexSignal{EstimatedRows: 100000, SeqScan: 150, IdxScan: 10},
			wantContains: []string{"smaller sample"},
		},
		{
			name:         "insufficient confidence mentions not enough data",
			signal:       IndexSignal{EstimatedRows: 500, SeqScan: 10000, IdxScan: 1},
			wantContains: []string{"Not enough data yet"},
		},
		{
			name:             "no suspected columns omits the filtering-on phrase",
			signal:           IndexSignal{EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10},
			suspectedColumns: nil,
			wantNotContains:  []string{"filtering on"},
		},
		{
			name: "poor selectivity mentions low selectivity favors-caution wording",
			signal: IndexSignal{
				EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10,
				NDistinct: floatPtr(4), // selectivity 0.25, above the 0.05 poor threshold
			},
			wantContains: []string{"low selectivity", "may not help much"},
		},
		{
			name: "good selectivity mentions the favors-an-index wording",
			signal: IndexSignal{
				EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10,
				NDistinct: floatPtr(90000), // selectivity ~0.0000111, well below 0.05
			},
			wantContains: []string{"looks selective", "favors an index"},
		},
		{
			name:               "zero existing indexes is called out explicitly",
			signal:             IndexSignal{EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10},
			existingIndexCount: 0,
			wantContains:       []string{"no indexes at all"},
		},
		{
			name:               "nonzero existing indexes omits the no-indexes wording",
			signal:             IndexSignal{EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10},
			existingIndexCount: 3,
			wantNotContains:    []string{"no indexes at all"},
		},
		{
			name:            "write volume above the threshold warns about overhead",
			signal:          IndexSignal{EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10},
			writesPerSecond: 5.0,
			wantContains:    []string{"writes/s", "add real overhead"},
		},
		{
			name:            "low write volume is called low-risk",
			signal:          IndexSignal{EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10},
			writesPerSecond: 0.5,
			wantContains:    []string{"Write volume looks low", "low-risk"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewIndexCandidate("orders", tt.signal, tt.suspectedColumns, tt.existingIndexCount, tt.writesPerSecond)

			for _, want := range tt.wantContains {
				if !contains(got.Rationale, want) {
					t.Errorf("Rationale = %q, want it to contain %q", got.Rationale, want)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if contains(got.Rationale, notWant) {
					t.Errorf("Rationale = %q, want it to NOT contain %q", got.Rationale, notWant)
				}
			}
		})
	}
}
