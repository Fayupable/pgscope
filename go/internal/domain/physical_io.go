package domain

import "fmt"

const MinExecReadsForKcacheWarning = 1000

// QueryPhysicalIO is the raw shape read from pg_stat_kcache, joined with
// pg_stat_statements for the query text — no judgment applied yet.
type QueryPhysicalIO struct {
	Query        string
	Calls        int64
	ExecReads    int64
	ExecWrites   int64
	UserTimeMs   float64
	SystemTimeMs float64
}

// PhysicalIOHotspot is a suggestion, never a certainty — high physical
// reads mean this query isn't being served from cache, which is a
// signal worth investigating, not a confirmed problem.
type PhysicalIOHotspot struct {
	Query        string  `json:"query"`
	Calls        int64   `json:"calls"`
	ExecReads    int64   `json:"execReads"`
	ExecWrites   int64   `json:"execWrites"`
	UserTimeMs   float64 `json:"userTimeMs"`
	SystemTimeMs float64 `json:"systemTimeMs"`
	Explanation  string  `json:"explanation"`
}

// DetectPhysicalIOHotspots filters query shapes down to the ones with a
// meaningful number of physical disk reads.
func DetectPhysicalIOHotspots(stats []QueryPhysicalIO) []PhysicalIOHotspot {
	result := make([]PhysicalIOHotspot, 0)
	for _, s := range stats {
		if s.ExecReads < MinExecReadsForKcacheWarning {
			continue
		}

		result = append(result, PhysicalIOHotspot{
			Query:        s.Query,
			Calls:        s.Calls,
			ExecReads:    s.ExecReads,
			ExecWrites:   s.ExecWrites,
			UserTimeMs:   s.UserTimeMs,
			SystemTimeMs: s.SystemTimeMs,
			Explanation: fmt.Sprintf(
				"This query shape has done %d physical disk block reads and %d writes across %d calls (%.1fms user CPU, %.1fms system CPU). High physical reads mean it isn't being served from cache — consider whether an index or more memory (shared_buffers) would help.",
				s.ExecReads, s.ExecWrites, s.Calls, s.UserTimeMs, s.SystemTimeMs,
			),
		})
	}
	return result
}
