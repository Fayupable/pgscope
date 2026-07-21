import type { Session } from '../../../shared/types/session'
import './SessionDetailPanel.css'

interface SessionDetailPanelProps {
    session: Session
    position: { x: number; y: number }
    onClose: () => void
}

export function SessionDetailPanel({ session, position, onClose }: SessionDetailPanelProps) {
    return (
        <div
            className="session-detail-panel"
            style={{ left: position.x, top: position.y }}
        >
            <div className="session-detail-panel__header">
                <span className="session-detail-panel__pid">PID {session.id}</span>
                <button className="session-detail-panel__close" onClick={onClose}>
                    ✕
                </button>
            </div>

            <dl className="session-detail-panel__meta">
                <div>
                    <dt>User</dt>
                    <dd>{session.user || 'unknown'}</dd>
                </div>
                <div>
                    <dt>Application</dt>
                    <dd>{session.applicationName || '—'}</dd>
                </div>
                <div>
                    <dt>State</dt>
                    <dd>{session.state}</dd>
                </div>
                <div>
                    <dt>Duration</dt>
                    <dd>{session.durationSeconds.toFixed(1)}s</dd>
                </div>
                <div>
                    <dt>Wait event</dt>
                    <dd>{session.waitEventType === 'none' ? '—' : `${session.waitEventType}: ${session.waitEvent}`}</dd>
                </div>
            </dl>

            <p className="session-detail-panel__query">{session.query}</p>

            {session.blockedBy.length > 0 && (
                <p className="session-detail-panel__blocked">
                    Waiting on: {session.blockedBy.join(', ')}
                </p>
            )}

            {session.locks.length > 0 && (
                <ul className="session-detail-panel__locks">
                    {session.locks.map((lock, index) => (
                        <li key={index} className="session-detail-panel__lock-row">
                            <span className={`dot dot--${lock.severity}`} />
                            <span className="session-detail-panel__lock-text">
                                {lock.resource ? `${lock.resource} · ` : ''}
                                {lock.nativeMode}
                            </span>
                        </li>
                    ))}
                </ul>
            )}
        </div>
    )
}