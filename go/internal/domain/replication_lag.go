package domain

import "fmt"

const ReplicationLagWarningBytes = 100 * 1024 * 1024 // 100 MiB

// ReplicaLagInfo is the raw shape read from pg_stat_replication for one
// connected replica — no judgment applied yet.
type ReplicaLagInfo struct {
	ApplicationName string
	ClientAddr      string
	LagBytes        int64
}

// ReplicationLagWarning is a suggestion, never a certainty — some lag
// is normal under load; this only flags replicas meaningfully behind.
type ReplicationLagWarning struct {
	ApplicationName string `json:"applicationName"`
	ClientAddr      string `json:"clientAddr"`
	LagBytes        int64  `json:"lagBytes"`
	Explanation     string `json:"explanation"`
}

// DetectReplicationLagWarnings filters connected replicas down to the
// ones far enough behind the primary to be worth a look.
func DetectReplicationLagWarnings(replicas []ReplicaLagInfo) []ReplicationLagWarning {
	result := make([]ReplicationLagWarning, 0)
	for _, r := range replicas {
		if r.LagBytes < ReplicationLagWarningBytes {
			continue
		}

		result = append(result, ReplicationLagWarning{
			ApplicationName: r.ApplicationName,
			ClientAddr:      r.ClientAddr,
			LagBytes:        r.LagBytes,
			Explanation: fmt.Sprintf(
				"Replica %q (%s) is about %d MB behind the primary. If this keeps growing, it risks falling further behind or WAL retention issues on the primary. Check its network connectivity and query load.",
				r.ApplicationName, r.ClientAddr, r.LagBytes/(1024*1024),
			),
		})
	}
	return result
}
