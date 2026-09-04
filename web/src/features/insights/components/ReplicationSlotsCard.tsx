import type { ReplicationSlotWarning } from '../../../shared/types/insights'
import { HealthCardHeader } from './HealthCardHeader'

export function ReplicationSlotsCard({ warnings }: { warnings: ReplicationSlotWarning[] }) {
    const hasIssues = warnings.length > 0

    return (
        <div className="health-card">
            <HealthCardHeader
                title="Replication slots"
                subtitle="Replicas that have fallen behind and are forcing extra data to be kept around."
                ok={!hasIssues}
            />

            {!hasIssues ? (
                <p className="insights-empty">No replication slots retaining excessive WAL.</p>
            ) : (
                <ul className="index-candidates-table__rationales">
                    {warnings.map((w) => (
                        <li key={w.slotName}>
                            <strong>{w.slotName}:</strong> {w.explanation}
                        </li>
                    ))}
                </ul>
            )}
        </div>
    )
}
