import type { InvalidIndex, UnvalidatedConstraint } from '../../../shared/types/insights'
import { HealthCardHeader } from './HealthCardHeader'

export function InvalidObjectsCard({
    invalidIndexes,
    unvalidatedConstraints,
}: {
    invalidIndexes: InvalidIndex[]
    unvalidatedConstraints: UnvalidatedConstraint[]
}) {
    const hasIssues = invalidIndexes.length > 0 || unvalidatedConstraints.length > 0

    return (
        <div className="health-card">
            <HealthCardHeader
                title="Invalid indexes & constraints"
                subtitle="Indexes or constraints left broken by an operation that didn't finish cleanly."
                ok={!hasIssues}
            />

            {!hasIssues ? (
                <p className="insights-empty">No invalid indexes or unvalidated constraints found.</p>
            ) : (
                <ul className="index-candidates-table__rationales">
                    {invalidIndexes.map((idx, i) => (
                        <li key={`idx-${i}`}>
                            <strong>{idx.index}:</strong> {idx.explanation}
                        </li>
                    ))}
                    {unvalidatedConstraints.map((c, i) => (
                        <li key={`con-${i}`}>
                            <strong>{c.constraint}:</strong> {c.explanation}
                        </li>
                    ))}
                </ul>
            )}
        </div>
    )
}