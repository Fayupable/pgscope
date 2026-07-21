import type { PaginationWarning } from '../../../shared/types/insights'

export function PaginationWarningsTable({
    warnings,
    nestedStatementsTracked,
}: {
    warnings: PaginationWarning[]
    nestedStatementsTracked: boolean
}) {
    return (
        <>
            {warnings.length === 0 ? (
                <p className="insights-empty">No deep-pagination patterns detected.</p>
            ) : (
                <ul className="index-candidates-table__rationales">
                    {warnings.map((w, i) => (
                        <li key={i}>
                            <strong className="pagination-warnings__query">{w.query}</strong>
                            <br />
                            {w.warning}
                        </li>
                    ))}
                </ul>
            )}

            {!nestedStatementsTracked && (
                <p className="insights-disabled-notice__hint">
                    Note: <code>pg_stat_statements.track</code> is currently <code>top</code> — queries run inside
                    functions, procedures, or <code>DO</code> blocks aren't tracked individually, so this list
                    can't see them. Set it to <code>all</code> yourself if you need full visibility; pgscope never
                    changes this setting.
                </p>
            )}
        </>
    )
}