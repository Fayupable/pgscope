import type { PhysicalIOHotspot } from '../../../shared/types/insights'

export function PhysicalIOHotspotsTable({
    hotspots,
    physicalIOEnabled,
}: {
    hotspots: PhysicalIOHotspot[]
    physicalIOEnabled: boolean
}) {
    if (!physicalIOEnabled) {
        return (
            <div className="insights-disabled-notice">
                <p>
                    Physical I/O tracking requires the <code>pg_stat_kcache</code> extension, which isn't installed
                    on this database.
                </p>
                <p>
                    pgscope never installs extensions itself. To enable it, run this yourself and restart
                    PostgreSQL (it must be added to <code>shared_preload_libraries</code>):
                </p>
                <pre className="insights-disabled-notice__code">CREATE EXTENSION IF NOT EXISTS pg_stat_kcache;</pre>
            </div>
        )
    }

    if (hotspots.length === 0) {
        return <p className="insights-empty">No physical I/O hotspots detected.</p>
    }

    return (
        <>
            <table className="index-candidates-table">
                <thead>
                    <tr>
                        <th>Query</th>
                        <th>Calls</th>
                        <th>Reads</th>
                        <th>Writes</th>
                        <th>User CPU</th>
                        <th>System CPU</th>
                    </tr>
                </thead>
                <tbody>
                    {hotspots.map((h, i) => (
                        <tr key={i}>
                            <td className="index-candidates-table__table">{h.query}</td>
                            <td>{h.calls.toLocaleString()}</td>
                            <td>{h.execReads.toLocaleString()}</td>
                            <td>{h.execWrites.toLocaleString()}</td>
                            <td>{h.userTimeMs.toFixed(1)} ms</td>
                            <td>{h.systemTimeMs.toFixed(1)} ms</td>
                        </tr>
                    ))}
                </tbody>
            </table>

            <ul className="index-candidates-table__rationales">
                {hotspots.map((h, i) => (
                    <li key={i}>
                        <strong>Query {i + 1}:</strong> {h.explanation}
                    </li>
                ))}
            </ul>
        </>
    )
}