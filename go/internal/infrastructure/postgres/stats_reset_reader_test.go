package postgres

import "testing"

func TestWritesPerSecond(t *testing.T) {
	tests := []struct {
		name           string
		writeOps       int64
		elapsedSeconds float64
		want           float64
	}{
		{
			name:           "normal rate calculation",
			writeOps:       100,
			elapsedSeconds: 10,
			want:           10,
		},
		{
			name:           "zero elapsed seconds avoids division by zero",
			writeOps:       100,
			elapsedSeconds: 0,
			want:           0,
		},
		{
			name:           "negative elapsed seconds is treated as unknown",
			writeOps:       100,
			elapsedSeconds: -5,
			want:           0,
		},
		{
			name:           "zero write ops yields zero rate",
			writeOps:       0,
			elapsedSeconds: 60,
			want:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := writesPerSecond(tt.writeOps, tt.elapsedSeconds); got != tt.want {
				t.Errorf("writesPerSecond(%d, %v) = %v, want %v", tt.writeOps, tt.elapsedSeconds, got, tt.want)
			}
		})
	}
}
