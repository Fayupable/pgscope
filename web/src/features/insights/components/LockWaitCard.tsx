import type { LockWaitWarning } from '../../../shared/types/insights'
import { HealthCardHeader } from './HealthCardHeader'

export function LockWaitCard({ warnings }: { warnings: LockWaitWarning[] }) {
    const hasIssues = warnings.length > 0

    return (
        <div className="health-card">
            <HealthCardHeader
                title="Lock waits"
                subtitle="Sessions stuck waiting for another session to release a lock."
                ok={!hasIssues}
            />

            {!hasIssues ? (
                <p className="insights-empty">No sessions waiting on a lock for too long.</p>
            ) : (
                <ul className="index-candidates-table__rationales">
                    {warnings.map((w) => (
                        <li key={w.waitingPid}>
                            <strong>
                                PID {w.waitingPid} blocked by PID {w.blockingPid} ({w.waitAgeSeconds.toFixed(0)}s):
                            </strong>{' '}
                            {w.explanation}
                        </li>
                    ))}
                </ul>
            )}
        </div>
    )
}
