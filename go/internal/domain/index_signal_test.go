package domain

import "testing"

func floatPtr(v float64) *float64 {
	return &v
}

func TestIndexSignal_Confidence(t *testing.T) {
	tests := []struct {
		name   string
		signal IndexSignal
		want   Confidence
	}{
		{
			name:   "small table never qualifies regardless of scan pattern",
			signal: IndexSignal{EstimatedRows: 500, SeqScan: 10000, IdxScan: 1},
			want:   ConfidenceInsufficient,
		},
		{
			name:   "index never used yields insufficient data, not a false strong signal",
			signal: IndexSignal{EstimatedRows: 100000, SeqScan: 10000, IdxScan: 0},
			want:   ConfidenceInsufficient,
		},
		{
			name:   "high index usage percentage disqualifies even with many seq scans",
			signal: IndexSignal{EstimatedRows: 100000, SeqScan: 100, IdxScan: 10000},
			want:   ConfidenceInsufficient,
		},
		{
			name:   "large table, high volume, dominant seq scan ratio is strong",
			signal: IndexSignal{EstimatedRows: 100000, SeqScan: 5000, IdxScan: 10},
			want:   ConfidenceStrong,
		},
		{
			name:   "qualifies but total scans below the strong-signal volume floor is weak",
			signal: IndexSignal{EstimatedRows: 100000, SeqScan: 150, IdxScan: 10},
			want:   ConfidenceWeak,
		},
		{
			name:   "qualifies but total scans below even the weak floor is insufficient",
			signal: IndexSignal{EstimatedRows: 100000, SeqScan: 50, IdxScan: 5},
			want:   ConfidenceInsufficient,
		},
		{
			name: "strong scan pattern demoted to weak when the suspected column has poor selectivity",
			signal: IndexSignal{
				EstimatedRows: 100000,
				SeqScan:       5000,
				IdxScan:       10,
				NDistinct:     floatPtr(4), // e.g. a 4-value status column
			},
			want: ConfidenceWeak,
		},
		{
			name: "strong scan pattern stays strong when the suspected column is highly selective",
			signal: IndexSignal{
				EstimatedRows: 100000,
				SeqScan:       5000,
				IdxScan:       10,
				NDistinct:     floatPtr(90000), // nearly unique
			},
			want: ConfidenceStrong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.signal.Confidence(); got != tt.want {
				t.Errorf("Confidence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexSignal_Selectivity(t *testing.T) {
	tests := []struct {
		name       string
		signal     IndexSignal
		wantKnown  bool
		wantApprox float64
	}{
		{
			name:      "unknown when NDistinct is nil",
			signal:    IndexSignal{EstimatedRows: 1000},
			wantKnown: false,
		},
		{
			name:      "unknown when estimated rows is zero",
			signal:    IndexSignal{EstimatedRows: 0, NDistinct: floatPtr(10)},
			wantKnown: false,
		},
		{
			name:       "positive n_distinct is used directly as the distinct count",
			signal:     IndexSignal{EstimatedRows: 1000, NDistinct: floatPtr(4)},
			wantKnown:  true,
			wantApprox: 0.25,
		},
		{
			name:       "negative n_distinct is a fraction of estimated rows",
			signal:     IndexSignal{EstimatedRows: 1000, NDistinct: floatPtr(-1)},
			wantKnown:  true,
			wantApprox: 0.001, // -1 means "as many distinct values as rows" => fully unique
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, known := tt.signal.Selectivity()
			if known != tt.wantKnown {
				t.Fatalf("Selectivity() known = %v, want %v", known, tt.wantKnown)
			}
			if known && (got < tt.wantApprox*0.99 || got > tt.wantApprox*1.01) {
				t.Errorf("Selectivity() = %v, want approx %v", got, tt.wantApprox)
			}
		})
	}
}

func TestIndexSignal_Qualifies(t *testing.T) {
	tests := []struct {
		name   string
		signal IndexSignal
		want   bool
	}{
		{
			name:   "meets every threshold",
			signal: IndexSignal{EstimatedRows: 5000, SeqScan: 100, IdxScan: 10},
			want:   true,
		},
		{
			name:   "too few rows",
			signal: IndexSignal{EstimatedRows: 999, SeqScan: 100, IdxScan: 10},
			want:   false,
		},
		{
			name:   "never used via an index at all",
			signal: IndexSignal{EstimatedRows: 5000, SeqScan: 100, IdxScan: 0},
			want:   false,
		},
		{
			name:   "index usage already at or above the ceiling",
			signal: IndexSignal{EstimatedRows: 5000, SeqScan: 5, IdxScan: 95},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.signal.Qualifies(); got != tt.want {
				t.Errorf("Qualifies() = %v, want %v", got, tt.want)
			}
		})
	}
}
