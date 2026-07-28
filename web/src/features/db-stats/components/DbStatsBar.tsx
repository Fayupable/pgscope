import { useDbStatsStream } from '../hooks/useDbStatsStream'
import './DbStatsBar.css'

export function DbStatsBar() {
    const stats = useDbStatsStream()

    return (
        <div className="db-stats-bar">
            <div className="db-stats-bar__metric">
                <span className="db-stats-bar__value">{stats.commitsPerSecond.toFixed(2)}</span>
                <span className="db-stats-bar__label">commits/s</span>
            </div>
            <div className="db-stats-bar__metric">
                <span className="db-stats-bar__value db-stats-bar__value--rollback">
                    {stats.rollbacksPerSecond.toFixed(2)}
                </span>
                <span className="db-stats-bar__label">rollbacks/s</span>
            </div>
            <div className="db-stats-bar__metric">
                <span
                    className={
                        'db-stats-bar__value' +
                        (stats.cacheHitRatio < 90 ? ' db-stats-bar__value--rollback' : '')
                    }
                >
                    {stats.cacheHitRatio.toFixed(1)}%
                </span>
                <span className="db-stats-bar__label">cache hit ratio</span>
            </div>
            <div className="db-stats-bar__metric">
                <span
                    className={
                        'db-stats-bar__value' +
                        (stats.tempBytesPerSecond > 1024 * 1024 ? ' db-stats-bar__value--rollback' : '')
                    }
                >
                    {formatBytesPerSecond(stats.tempBytesPerSecond)}
                </span>
                <span className="db-stats-bar__label">temp files/s</span>
            </div>
        </div>
    )
}

function formatBytesPerSecond(bytesPerSecond: number): string {
    if (bytesPerSecond < 1024) return `${bytesPerSecond.toFixed(0)} B/s`
    if (bytesPerSecond < 1024 * 1024) return `${(bytesPerSecond / 1024).toFixed(1)} KB/s`
    return `${(bytesPerSecond / (1024 * 1024)).toFixed(1)} MB/s`
}