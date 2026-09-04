import type { ConnectionSaturation } from '../../../shared/types/insights'
import { HealthCardHeader } from './HealthCardHeader'

export function ConnectionSaturationCard({ saturation }: { saturation: ConnectionSaturation }) {
    const isWarning = Boolean(saturation.warning)

    return (
        <div className="health-card">
            <HealthCardHeader
                title="Connections"
                subtitle="How many of the database's connection slots are currently in use."
                ok={!isWarning}
            />
            <p className="health-card__value">
                {saturation.activeConnections} / {saturation.maxConnections}
                <span className="health-card__value-suffix"> ({saturation.usagePercent.toFixed(0)}%)</span>
            </p>
            {saturation.warning && <p className="health-card__warning">{saturation.warning}</p>}
        </div>
    )
}