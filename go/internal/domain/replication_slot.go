package domain

import "fmt"

const ReplicationSlotWarningBytes = 1024 * 1024 * 1024 // 1 GiB

// ReplicationSlotInfo is the raw shape read from pg_replication_slots for
// one replication slot — no judgment applied yet. RetainedBytes is how much
// WAL the primary is holding onto specifically because of this slot,
// regardless of whether a replica is currently connected to consume it.
type ReplicationSlotInfo struct {
	SlotName      string
	Active        bool
	RetainedBytes int64
}

// ReplicationSlotWarning is a suggestion, never a certainty — a slot
// retaining a lot of WAL is often transient (a replica catching up after a
// brief disconnect). This only flags slots holding onto enough WAL that,
// left unaddressed, risk filling the primary's disk — a different failure
// mode than replication lag itself (a slot with no connected replica at
// all still retains WAL, and wouldn't show up as "lag" anywhere).
type ReplicationSlotWarning struct {
	SlotName      string `json:"slotName"`
	Active        bool   `json:"active"`
	RetainedBytes int64  `json:"retainedBytes"`
	Explanation   string `json:"explanation"`
}

// DetectReplicationSlotWarnings filters replication slots down to the ones
// retaining enough WAL to be worth a look.
func DetectReplicationSlotWarnings(slots []ReplicationSlotInfo) []ReplicationSlotWarning {
	result := make([]ReplicationSlotWarning, 0)
	for _, s := range slots {
		if s.RetainedBytes < ReplicationSlotWarningBytes {
			continue
		}

		result = append(result, ReplicationSlotWarning{
			SlotName:      s.SlotName,
			Active:        s.Active,
			RetainedBytes: s.RetainedBytes,
			Explanation:   buildReplicationSlotExplanation(s),
		})
	}
	return result
}

func buildReplicationSlotExplanation(s ReplicationSlotInfo) string {
	retainedMB := s.RetainedBytes / (1024 * 1024)
	if !s.Active {
		return fmt.Sprintf(
			"Replication slot %q is inactive (no replica currently connected) but is still retaining about %d MB of WAL. An inactive slot retains WAL indefinitely until either a replica consumes it or the slot is dropped — this can fill the primary's disk if left unaddressed. Confirm the slot is still needed before dropping it.",
			s.SlotName, retainedMB,
		)
	}
	return fmt.Sprintf(
		"Replication slot %q is retaining about %d MB of WAL for its replica. If this keeps growing, the replica may be falling behind or struggling to keep up — check its connectivity and load.",
		s.SlotName, retainedMB,
	)
}
