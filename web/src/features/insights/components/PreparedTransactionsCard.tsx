import type { PreparedTransactionWarning } from '../../../shared/types/insights'
import { HealthCardHeader } from './HealthCardHeader'

export function PreparedTransactionsCard({ warnings }: { warnings: PreparedTransactionWarning[] }) {
    const hasIssues = warnings.length > 0

    return (
        <div className="health-card">
            <HealthCardHeader
                title="Prepared transactions"
                subtitle="Two-phase-commit transactions stuck waiting to finish."
                ok={!hasIssues}
            />

            {!hasIssues ? (
                <p className="insights-empty">No orphaned prepared transactions.</p>
            ) : (
                <ul className="index-candidates-table__rationales">
                    {warnings.map((w) => (
                        <li key={w.gid}>
                            <strong>{w.gid}:</strong> {w.explanation}
                        </li>
                    ))}
                </ul>
            )}
        </div>
    )
}
