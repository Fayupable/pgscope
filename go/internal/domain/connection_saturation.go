package domain

import "fmt"

const ConnectionSaturationWarningPercent = 80.0

// ConnectionSaturation is a plain descriptive fact with an attached
// warning only when usage crosses the threshold — like every other
// advisory here, it's informational, not a command.
type ConnectionSaturation struct {
	ActiveConnections int     `json:"activeConnections"`
	MaxConnections    int     `json:"maxConnections"`
	UsagePercent      float64 `json:"usagePercent"`
	Warning           string  `json:"warning,omitempty"`
}

// NewConnectionSaturation computes usage percentage and, if it's high
// enough to matter, a warning message. All judgment lives here so the
// infrastructure adapter stays pure I/O.
func NewConnectionSaturation(active, max int) ConnectionSaturation {
	usage := 0.0
	if max > 0 {
		usage = 100 * float64(active) / float64(max)
	}

	warning := ""
	if usage >= ConnectionSaturationWarningPercent {
		warning = fmt.Sprintf(
			"Using %d of %d max connections (%.0f%%) — approaching the configured limit. New connection attempts may start failing once max_connections is reached.",
			active, max, usage,
		)
	}

	return ConnectionSaturation{
		ActiveConnections: active,
		MaxConnections:    max,
		UsagePercent:      usage,
		Warning:           warning,
	}
}
