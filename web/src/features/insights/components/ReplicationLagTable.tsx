import type { ReplicationLagWarning } from '../../../shared/types/insights'

export function ReplicationLagTable({ warnings }: { warnings: ReplicationLagWarning[] }) {
    if (warnings.length === 0) {
        return <p className="insights-empty">No replicas meaningfully behind the primary.</p>
    }

    return (
        <>
            <table className="index-candidates-table">
                <thead>
                    <tr>
                        <th>Application</th>
                        <th>Client address</th>
                        <th>Lag</th>
                    </tr>
                </thead>
                <tbody>
                    {warnings.map((w, i) => (
                        <tr key={i}>
                            <td className="index-candidates-table__table">{w.applicationName}</td>
                            <td>{w.clientAddr}</td>
                            <td>{(w.lagBytes / (1024 * 1024)).toFixed(0)} MB</td>
                        </tr>
                    ))}
                </tbody>
            </table>

            <ul className="index-candidates-table__rationales">
                {warnings.map((w, i) => (
                    <li key={i}>
                        <strong>{w.applicationName}:</strong> {w.explanation}
                    </li>
                ))}
            </ul>
        </>
    )
}