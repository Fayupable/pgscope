import type { UnloggedTable } from '../../../shared/types/insights'
import { HealthCardHeader } from './HealthCardHeader'

export function UnloggedTablesCard({ tables }: { tables: UnloggedTable[] }) {
    const hasIssues = tables.length > 0

    return (
        <div className="health-card">
            <HealthCardHeader
                title="Unlogged tables"
                subtitle="Tables whose data is lost on a crash or restart, by design."
                ok={!hasIssues}
            />

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
