import type { CheckpointHealth } from '../../../shared/types/insights'
import { HealthCardHeader } from './HealthCardHeader'

export function CheckpointHealthCard({ health }: { health: CheckpointHealth }) {
    const isWarning = Boolean(health.warning)

    return (
        <div className="health-card">
            <HealthCardHeader
                title="Checkpoints"
                subtitle="How often disk writes get forced early instead of on their normal schedule."
                ok={!isWarning}
            />
            <p className="health-card__value">
                {health.requestedCheckpoints} / {health.scheduledCheckpoints + health.requestedCheckpoints}
                <span className="health-card__value-suffix"> forced ({health.requestedRatio.toFixed(1)}%)</span>
            </p>
            {health.warning && <p className="health-card__warning">{health.warning}</p>}
        </div>
    )
}