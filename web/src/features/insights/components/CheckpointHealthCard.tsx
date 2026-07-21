import type { CheckpointHealth } from '../../../shared/types/insights'

export function CheckpointHealthCard({ health }: { health: CheckpointHealth }) {
    const isWarning = Boolean(health.warning)

    return (
        <div className="health-card">
            <div className="health-card__header">
                <span className={'health-card__status' + (isWarning ? ' health-card__status--warning' : ' health-card__status--ok')}>
                    {isWarning ? '⚠' : '✓'}
                </span>
                <span className="health-card__title">Checkpoints</span>
            </div>
            <p className="health-card__value">
                {health.requestedCheckpoints} / {health.scheduledCheckpoints + health.requestedCheckpoints}
                <span className="health-card__value-suffix"> forced ({health.requestedRatio.toFixed(1)}%)</span>
            </p>
            {health.warning && <p className="health-card__warning">{health.warning}</p>}
        </div>
    )
}