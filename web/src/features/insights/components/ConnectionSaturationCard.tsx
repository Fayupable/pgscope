import type { ConnectionSaturation } from '../../../shared/types/insights'

export function ConnectionSaturationCard({ saturation }: { saturation: ConnectionSaturation }) {
    const isWarning = Boolean(saturation.warning)

    return (
        <div className="health-card">
            <div className="health-card__header">
                <span className={'health-card__status' + (isWarning ? ' health-card__status--warning' : ' health-card__status--ok')}>
                    {isWarning ? '⚠' : '✓'}
                </span>
                <span className="health-card__title">Connections</span>
            </div>
            <p className="health-card__value">
                {saturation.activeConnections} / {saturation.maxConnections}
                <span className="health-card__value-suffix"> ({saturation.usagePercent.toFixed(0)}%)</span>
            </p>
            {saturation.warning && <p className="health-card__warning">{saturation.warning}</p>}
        </div>
    )
}