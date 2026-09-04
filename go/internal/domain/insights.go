package domain

// SlowQuery represents one normalized query's aggregate cost, as reported
// by pg_stat_statements — not a single execution, but the cumulative cost
// across every time this query shape has run.
type SlowQuery struct {
	Query       string  `json:"query"`
	Calls       int64   `json:"calls"`
	TotalExecMs float64 `json:"totalExecMs"`
	MeanExecMs  float64 `json:"meanExecMs"`
}

// IndexCandidate is a suggestion, never a certainty — the caller must
// present it as something to evaluate, not something to blindly apply.
// It always carries both the potential read benefit and the write-side
// cost of adding an index, since one without the other is misleading.
type IndexCandidate struct {
	Table              string     `json:"table"`
	SeqScanCount       int64      `json:"seqScanCount"`
	IdxScanCount       int64      `json:"idxScanCount"`
	SuspectedColumns   []string   `json:"suspectedColumns"`
	ExistingIndexCount int        `json:"existingIndexCount"`
	WritesPerSecond    float64    `json:"writesPerSecond"`
	Confidence         Confidence `json:"confidence"`
	Rationale          string     `json:"rationale"`
	Selectivity        *float64   `json:"selectivity,omitempty"`
}

// Insights bundles every category of suggestion returned by one snapshot
// of analysis. Always advisory — pgscope never applies anything itself.
type Insights struct {
	TopQueries                  []SlowQuery                  `json:"topQueries"`
	IndexCandidates             []IndexCandidate             `json:"indexCandidates"`
	DuplicateIndexes            []DuplicateIndex             `json:"duplicateIndexes"`
	UnusedIndexes               []UnusedIndex                `json:"unusedIndexes"`
	FunctionCosts               []FunctionCost               `json:"functionCosts"`
	TrackFunctionsEnabled       bool                         `json:"trackFunctionsEnabled"`
	TrackFunctionsSetting       string                       `json:"trackFunctionsSetting"`
	PaginationWarnings          []PaginationWarning          `json:"paginationWarnings"`
	NestedStatementsTracked     bool                         `json:"nestedStatementsTracked"`
	StatementsTrackSetting      string                       `json:"statementsTrackSetting"`
	DatabaseSize                DatabaseSizeInfo             `json:"databaseSize"`
	ConnectionSaturation        ConnectionSaturation         `json:"connectionSaturation"`
	SequenceOverflowRisks       []SequenceOverflowRisk       `json:"sequenceOverflowRisks"`
	InvalidIndexes              []InvalidIndex               `json:"invalidIndexes"`
	UnvalidatedConstraints      []UnvalidatedConstraint      `json:"unvalidatedConstraints"`
	VacuumHealthWarnings        []VacuumHealthWarning        `json:"vacuumHealthWarnings"`
	IdleInTransactionWarnings   []IdleInTransactionWarning   `json:"idleInTransactionWarnings"`
	CheckpointHealth            CheckpointHealth             `json:"checkpointHealth"`
	ReplicationLagWarnings      []ReplicationLagWarning      `json:"replicationLagWarnings"`
	PhysicalIOEnabled           bool                         `json:"physicalIOEnabled"`
	PhysicalIOHotspots          []PhysicalIOHotspot          `json:"physicalIOHotspots"`
	PreparedTransactionWarnings []PreparedTransactionWarning `json:"preparedTransactionWarnings"`
	ReplicationSlotWarnings     []ReplicationSlotWarning     `json:"replicationSlotWarnings"`
	LongRunningQueryWarnings    []LongRunningQueryWarning    `json:"longRunningQueryWarnings"`
	UnloggedTables              []UnloggedTable              `json:"unloggedTables"`
	// LockWaitWarnings is MySQL-only for now — see domain.LockWaitSession.
	LockWaitWarnings []LockWaitWarning `json:"lockWaitWarnings"`
}
