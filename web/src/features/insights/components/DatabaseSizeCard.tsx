import { formatBytes } from '../utils/formatBytes'

export function DatabaseSizeCard({ totalBytes }: { totalBytes: number }) {
    return (
        <div className="health-card">
            <div className="health-card__header">
                <span className="health-card__status health-card__status--ok">✓</span>
                <span className="health-card__title">Database size</span>
            </div>
            <p className="health-card__value">{formatBytes(totalBytes)}</p>
        </div>
    )
}