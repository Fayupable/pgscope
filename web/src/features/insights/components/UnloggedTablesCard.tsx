import type { UnloggedTable } from '../../../shared/types/insights'

export function UnloggedTablesCard({ tables }: { tables: UnloggedTable[] }) {
    const hasIssues = tables.length > 0

    return (
        <div className="health-card">
            <div className="health-card__header">
                <span className={'health-card__status' + (hasIssues ? ' health-card__status--warning' : ' health-card__status--ok')}>
                    {hasIssues ? '⚠' : '✓'}
                </span>
                <span className="health-card__title">Unlogged tables</span>
            </div>

            {!hasIssues ? (
                <p className="insights-empty">No unlogged tables found.</p>
            ) : (
                <ul className="index-candidates-table__rationales">
                    {tables.map((t) => (
                        <li key={t.table}>
                            <strong>{t.table}:</strong> {t.explanation}
                        </li>
                    ))}
                </ul>
            )}
        </div>
    )
}
