package domain

// TableSize describes one table's total on-disk footprint (heap +
// indexes + TOAST) — a purely descriptive fact, no judgment involved.
type TableSize struct {
	Table      string `json:"table"`
	TotalBytes int64  `json:"totalBytes"`
}

// DatabaseSizeInfo summarizes overall storage usage — the total
// database size plus its largest tables, so a DBA can see at a glance
// where the space is actually going.
type DatabaseSizeInfo struct {
	TotalBytes    int64       `json:"totalBytes"`
	LargestTables []TableSize `json:"largestTables"`
}
