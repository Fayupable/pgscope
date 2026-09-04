import type { SequenceOverflowRisk } from '../../../shared/types/insights'
import { HealthCardHeader } from './HealthCardHeader'

const SUBTITLE = 'Auto-incrementing IDs getting close to their maximum value.'

export function SequenceOverflowTable({ risks }: { risks: SequenceOverflowRisk[] }) {
    if (risks.length === 0) {
        return (
            <div className="health-card">
                <HealthCardHeader title="Sequence overflow risk" subtitle={SUBTITLE} ok />
                <p className="insights-empty">No sequences approaching their maximum value.</p>
            </div>
        )
    }

    return (
        <div className="health-card">
            <HealthCardHeader title="Sequence overflow risk" subtitle={SUBTITLE} ok={false} />
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