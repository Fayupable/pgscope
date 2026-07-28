package domain

import "testing"

func TestNewConnectionSaturation(t *testing.T) {
	tests := []struct {
		name        string
		active      int
		max         int
		wantUsage   float64
		wantWarning bool
	}{
		{
			name:        "max is zero yields zero usage, not division by zero",
			active:      10,
			max:         0,
			wantUsage:   0,
			wantWarning: false,
		},
		{
			name:        "no active connections yields zero usage",
			active:      0,
			max:         100,
			wantUsage:   0,
			wantWarning: false,
		},
		{
			name:        "below the warning threshold has usage but no warning",
			active:      50,
			max:         100,
			wantUsage:   50,
			wantWarning: false,
		},
		{
			name:        "exactly at the warning threshold triggers a warning",
			active:      80,
			max:         100,
			wantUsage:   80,
			wantWarning: true,
		},
		{
			name:        "above the warning threshold triggers a warning",
			active:      95,
			max:         100,
			wantUsage:   95,
			wantWarning: true,
		},
		{
			name:        "fully saturated is one hundred percent",
			active:      100,
			max:         100,
			wantUsage:   100,
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewConnectionSaturation(tt.active, tt.max)

			if got.UsagePercent != tt.wantUsage {
				t.Errorf("UsagePercent = %v, want %v", got.UsagePercent, tt.wantUsage)
			}
			hasWarning := got.Warning != ""
			if hasWarning != tt.wantWarning {
				t.Errorf("Warning present = %v (%q), want present = %v", hasWarning, got.Warning, tt.wantWarning)
			}
			if got.ActiveConnections != tt.active {
				t.Errorf("ActiveConnections = %d, want %d", got.ActiveConnections, tt.active)
			}
			if got.MaxConnections != tt.max {
				t.Errorf("MaxConnections = %d, want %d", got.MaxConnections, tt.max)
			}
		})
	}
}

func TestNewConnectionSaturation_WarningContent(t *testing.T) {
	got := NewConnectionSaturation(90, 100)

	for _, want := range []string{"90 of 100", "90%", "max_connections"} {
		if !contains(got.Warning, want) {
			t.Errorf("Warning = %q, want it to contain %q", got.Warning, want)
		}
	}
}
