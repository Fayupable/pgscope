package domain

import "fmt"

const (
	MaxScansForUnused           = 50
	MinStatsAgeSecondsForUnused = 3600 // give the database at least an hour of observed activity before judging
)

// UnusedIndexInfo is the raw shape read from the system catalog for one
// non-unique index — no judgment applied yet.
type UnusedIndexInfo struct {
	Table      string
	Index      string
	SizeBytes  int64
	IndexScans int64
}

// UnusedIndex is a suggestion, never a certainty — same philosophy as
// every other advisory type in this package. A low scan count over a
// short observation window means "not enough data," not "definitely
// unused," so the caller must know how long stats have been accumulating.
type UnusedIndex struct {
	Table       string `json:"table"`
	Index       string `json:"index"`
	SizeBytes   int64  `json:"sizeBytes"`
	IndexScans  int64  `json:"indexScans"`
	Explanation string `json:"explanation"`
}

// DetectUnusedIndexes flags indexes with very few scans, but only once
// statistics have been accumulating long enough for a low count to mean
// something — mirrors pgHero's proven idx_scan threshold, with an added
// time-based gate this project's other advisories already apply.
func DetectUnusedIndexes(indexes []UnusedIndexInfo, statsAgeSeconds float64) []UnusedIndex {
	if statsAgeSeconds < MinStatsAgeSecondsForUnused {
		return []UnusedIndex{}
	}

	result := make([]UnusedIndex, 0, len(indexes))
	for _, idx := range indexes {
		if idx.IndexScans > MaxScansForUnused {
			continue
		}
		result = append(result, UnusedIndex{
			Table:       idx.Table,
			Index:       idx.Index,
			SizeBytes:   idx.SizeBytes,
			IndexScans:  idx.IndexScans,
			Explanation: buildUnusedExplanation(idx, statsAgeSeconds),
		})
	}
	return result
}

func buildUnusedExplanation(idx UnusedIndexInfo, statsAgeSeconds float64) string {
	hours := statsAgeSeconds / 3600
	return fmt.Sprintf(
		"Index %q on %s has been scanned only %d time(s) over roughly %.1f hours of observed activity, and occupies %s. It may be safe to drop, but verify it isn't used by a rare batch job or reporting query before doing so.",
		idx.Index, idx.Table, idx.IndexScans, hours, formatBytes(idx.SizeBytes),
	)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
