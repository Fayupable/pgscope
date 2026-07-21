import type { VacuumHealthWarning } from '../../../shared/types/insights'

export function VacuumHealthTable({ warnings }: { warnings: VacuumHealthWarning[] }) {
    if (warnings.length === 0) {
        return <p className="insights-empty">No tables with a high dead-tuple ratio.</p>
    }

    return (
        <table className="index-candidates-table">
            <thead>
                <tr>
                    <th>Table</th>
                    <th>Dead tuple ratio</th>
                    <th>Last autovacuum</th>
                </tr>
            </thead>
            <tbody>
                {warnings.map((w, i) => (
                    <tr key={i}>
                        <td className="index-candidates-table__table">{w.table}</td>
                        <td>{w.deadTupleRatio.toFixed(0)}%</td>
                        <td>{w.lastAutovacuum ? new Date(w.lastAutovacuum).toLocaleString() : 'never'}</td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}