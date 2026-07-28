import type { LongRunningQueryWarning } from '../../../shared/types/insights'

export function LongRunningQueriesCard({ warnings }: { warnings: LongRunningQueryWarning[] }) {
    const hasIssues = warnings.length > 0

    return (
        <div className="health-card">
            <div className="health-card__header">
                <span className={'health-card__status' + (hasIssues ? ' health-card__status--warning' : ' health-card__status--ok')}>
                    {hasIssues ? '⚠' : '✓'}
                </span>
                <span className="health-card__title">Long-running queries</span>
            </div>

            {!hasIssues ? (
                <p className="insights-empty">No long-running active queries.</p>
            ) : (
                <ul className="index-candidates-table__rationales">
                    {warnings.map((w) => (
                        <li key={w.pid}>
                            <strong>PID {w.pid} ({w.runningSeconds.toFixed(0)}s):</strong> {w.explanation}
                        </li>
                    ))}
                </ul>
            )}
        </div>
    )
}
