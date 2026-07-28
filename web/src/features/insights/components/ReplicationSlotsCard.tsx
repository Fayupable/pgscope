import type { ReplicationSlotWarning } from '../../../shared/types/insights'

export function ReplicationSlotsCard({ warnings }: { warnings: ReplicationSlotWarning[] }) {
    const hasIssues = warnings.length > 0

    return (
        <div className="health-card">
            <div className="health-card__header">
                <span className={'health-card__status' + (hasIssues ? ' health-card__status--warning' : ' health-card__status--ok')}>
                    {hasIssues ? '⚠' : '✓'}
                </span>
                <span className="health-card__title">Replication slots</span>
            </div>

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
