import type { FunctionCost } from '../../../shared/types/insights'
import { formatDuration } from '../utils/formatDuration'

export function FunctionCostsTable({
    functions,
    trackFunctionsEnabled,
}: {
    functions: FunctionCost[]
    trackFunctionsEnabled: boolean
}) {
    if (!trackFunctionsEnabled) {
        return (
            <div className="insights-disabled-notice">
                <p>
                    Function and trigger cost tracking requires <code>track_functions</code> to be set to{' '}
                    <code>pl</code> (or <code>all</code>) — it's <code>none</code> by default.
                </p>
                <p>
                    Enabling it has a small, ongoing per-call overhead. We recommend <code>pl</code> over{' '}
                    <code>all</code> — it covers PL/pgSQL functions and triggers (the overwhelming majority of
                    real-world trigger code) at proportionally lower cost, since <code>all</code> also profiles
                    fast built-in C functions where the same fixed overhead is proportionally more expensive.
                </p>
                <p>
                    pgscope never changes this setting itself. To enable it, run this yourself and reload
                    PostgreSQL:
                </p>
                <pre className="insights-disabled-notice__code">ALTER SYSTEM SET track_functions = 'pl';{'\n'}SELECT pg_reload_conf();</pre>
            </div>
        )
    }

    if (functions.length === 0) {
        return <p className="insights-empty">No expensive functions or triggers detected.</p>
    }

    return (
        <>
            <table className="index-candidates-table">
                <thead>
                    <tr>
                        <th>Function</th>
                        <th>Trigger on</th>
                        <th>Calls</th>
                        <th>Self-time</th>
                    </tr>
                </thead>
                <tbody>
                    {functions.map((f, i) => (
                        <tr key={i}>
                            <td className="index-candidates-table__table">{f.function}</td>
                            <td>{f.isTrigger ? (f.triggerTables ?? []).join(', ') : <span className="insights-empty">—</span>}</td>
                            <td>{f.calls.toLocaleString()}</td>
                            <td>{formatDuration(f.selfTimeMs)}</td>
                        </tr>
                    ))}
                </tbody>
            </table>

            <ul className="index-candidates-table__rationales">
                {functions.map((f, i) => (
                    <li key={i}>
                        <strong>{f.function}:</strong> {f.explanation}
                    </li>
                ))}
            </ul>
        </>
    )
}