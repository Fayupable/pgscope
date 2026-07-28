import type { PreparedTransactionWarning } from '../../../shared/types/insights'

export function PreparedTransactionsCard({ warnings }: { warnings: PreparedTransactionWarning[] }) {
    const hasIssues = warnings.length > 0

    return (
        <div className="health-card">
            <div className="health-card__header">
                <span className={'health-card__status' + (hasIssues ? ' health-card__status--warning' : ' health-card__status--ok')}>
                    {hasIssues ? '⚠' : '✓'}
                </span>
                <span className="health-card__title">Prepared transactions</span>
            </div>

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
