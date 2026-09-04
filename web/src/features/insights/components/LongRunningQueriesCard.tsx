import type { LongRunningQueryWarning } from '../../../shared/types/insights'
import { HealthCardHeader } from './HealthCardHeader'

export function LongRunningQueriesCard({ warnings }: { warnings: LongRunningQueryWarning[] }) {
    const hasIssues = warnings.length > 0

    return (
        <div className="health-card">
            <HealthCardHeader
                title="Long-running queries"
                subtitle="Queries that have been actively running far longer than normal."
                ok={!hasIssues}
            />

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
