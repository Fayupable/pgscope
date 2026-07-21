import type { SequenceOverflowRisk } from '../../../shared/types/insights'

export function SequenceOverflowTable({ risks }: { risks: SequenceOverflowRisk[] }) {
    if (risks.length === 0) {
        return (
            <div className="health-card">
                <div className="health-card__header">
                    <span className="health-card__status health-card__status--ok">✓</span>
                    <span className="health-card__title">Sequence overflow risk</span>
                </div>
                <p className="insights-empty">No sequences approaching their maximum value.</p>
            </div>
        )
    }

    return (
        <div className="health-card">
            <div className="health-card__header">
                <span className="health-card__status health-card__status--warning">⚠</span>
                <span className="health-card__title">Sequence overflow risk</span>
            </div>
            <ul className="index-candidates-table__rationales">
                {risks.map((r, i) => (
                    <li key={i}>
                        <strong>{r.sequence}:</strong> {r.explanation}
                    </li>
                ))}
            </ul>
        </div>
    )
}