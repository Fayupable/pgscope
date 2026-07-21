import type { IdleInTransactionWarning } from '../../../shared/types/insights'

export function IdleInTransactionTable({ warnings }: { warnings: IdleInTransactionWarning[] }) {
    if (warnings.length === 0) {
        return <p className="insights-empty">No sessions idle in a transaction for too long.</p>
    }

    return (
        <table className="index-candidates-table">
            <thead>
                <tr>
                    <th>PID</th>
                    <th>User</th>
                    <th>Application</th>
                    <th>Idle for</th>
                </tr>
            </thead>
            <tbody>
                {warnings.map((w) => (
                    <tr key={w.pid}>
                        <td className="index-candidates-table__table">{w.pid}</td>
                        <td>{w.user}</td>
                        <td>{w.applicationName || <span className="insights-empty">—</span>}</td>
                        <td>{w.idleSeconds.toFixed(0)}s</td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}