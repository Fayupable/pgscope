package domain

import "testing"

func TestPaginationSignal_Suspected(t *testing.T) {
	tests := []struct {
		name   string
		signal PaginationSignal
		want   bool
	}{
		{
			name: "meets every threshold",
			signal: PaginationSignal{
				ContainsOffset: true,
				Calls:          500,
				MeanExecMs:     20,
				StddevExecMs:   30,
				Rows:           5000, // 10 rows/call
			},
			want: true,
		},
		{
			name: "no OFFSET in query text at all",
			signal: PaginationSignal{
				ContainsOffset: false,
				Calls:          500,
				MeanExecMs:     20,
				StddevExecMs:   30,
				Rows:           5000,
			},
			want: false,
		},
		{
			name: "too few calls to trust the statistics",
			signal: PaginationSignal{
				ContainsOffset: true,
				Calls:          10,
				MeanExecMs:     20,
				StddevExecMs:   30,
				Rows:           100,
			},
			want: false,
		},
		{
			name: "already fast, not worth warning about",
			signal: PaginationSignal{
				ContainsOffset: true,
				Calls:          500,
				MeanExecMs:     1,
				StddevExecMs:   2,
				Rows:           5000,
			},
			want: false,
		},
		{
			name: "large average row count looks like a report, not pagination",
			signal: PaginationSignal{
				ContainsOffset: true,
				Calls:          500,
				MeanExecMs:     20,
				StddevExecMs:   30,
				Rows:           500000, // 1000 rows/call
			},
			want: false,
		},
		{
			name: "low variance means offset likely isn't varying much across calls",
			signal: PaginationSignal{
				ContainsOffset: true,
				Calls:          500,
				MeanExecMs:     20,
				StddevExecMs:   5, // CV = 0.25
				Rows:           5000,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.signal.Suspected(); got != tt.want {
				t.Errorf("Suspected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaginationSignal_Warning(t *testing.T) {
	t.Run("empty when not suspected", func(t *testing.T) {
		signal := PaginationSignal{ContainsOffset: false}
		if got := signal.Warning(); got != "" {
			t.Errorf("Warning() = %q, want empty string", got)
		}
	})

	t.Run("non-empty and mentions inference when suspected", func(t *testing.T) {
		signal := PaginationSignal{
			ContainsOffset: true,
			Calls:          500,
			MeanExecMs:     20,
			StddevExecMs:   30,
			Rows:           5000,
		}
		got := signal.Warning()
		if got == "" {
			t.Fatal("Warning() = empty, want a non-empty message")
		}
		if !contains(got, "inference") {
			t.Errorf("Warning() = %q, want it to mention this is an inference", got)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
