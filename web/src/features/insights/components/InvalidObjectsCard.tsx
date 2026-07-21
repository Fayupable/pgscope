import type { InvalidIndex, UnvalidatedConstraint } from '../../../shared/types/insights'

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
            <div className="health-card__header">
                <span className={'health-card__status' + (hasIssues ? ' health-card__status--warning' : ' health-card__status--ok')}>
                    {hasIssues ? '⚠' : '✓'}
                </span>
                <span className="health-card__title">Invalid indexes & constraints</span>
            </div>

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