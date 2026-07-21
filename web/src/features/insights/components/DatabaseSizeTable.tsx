import type { TableSize } from '../../../shared/types/insights'
import { formatBytes } from '../utils/formatBytes'

export function DatabaseSizeTable({ tables }: { tables: TableSize[] }) {
    if (tables.length === 0) {
        return <p className="insights-empty">No tables found.</p>
    }

    return (
        <table className="index-candidates-table">
            <thead>
                <tr>
                    <th>Table</th>
                    <th>Total size</th>
                </tr>
            </thead>
            <tbody>
                {tables.map((t) => (
                    <tr key={t.table}>
                        <td className="index-candidates-table__table">{t.table}</td>
                        <td>{formatBytes(t.totalBytes)}</td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}