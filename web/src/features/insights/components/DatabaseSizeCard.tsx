import { formatBytes } from '../utils/formatBytes'
import { HealthCardHeader } from './HealthCardHeader'

export function DatabaseSizeCard({ totalBytes }: { totalBytes: number }) {
    return (
        <div className="health-card">
            <HealthCardHeader title="Database size" subtitle="Total size of this database and its largest tables." ok />
            <p className="health-card__value">{formatBytes(totalBytes)}</p>
        </div>
    )
}