package domain

import "time"

// SnapshotTrigger describes why a Snapshot was captured — a routine
// periodic sample, or an event worth keeping longer (e.g. a new blocking
// relationship appeared).
type SnapshotTrigger string

const (
	SnapshotTriggerPeriodic SnapshotTrigger = "periodic"
	SnapshotTriggerIncident SnapshotTrigger = "incident"
)

// Snapshot is one recorded moment of session activity, used to build a
// rolling history and to support exporting/replaying past activity.
type Snapshot struct {
	Sessions   []Session       `json:"sessions"`
	CapturedAt time.Time       `json:"capturedAt"`
	Trigger    SnapshotTrigger `json:"trigger"`
}
