import type { Session, LockSeverity } from '../../../shared/types/session'
import { getSessionStatus } from '../../../shared/utils/sessionStatus'
import './SessionTable.css'

const SEVERITY_RANK: Record<LockSeverity, number> = {
    exclusive: 4,
    intent: 3,
    shared_write: 2,
    shared_read: 1,
    unknown: 0,
}

function formatDuration(seconds: number): string {
    const totalSeconds = Math.floor(seconds)
    const minutes = Math.floor(totalSeconds / 60)
    const remaining = totalSeconds % 60
    return `${String(minutes).padStart(2, '0')}:${String(remaining).padStart(2, '0')}`
}

function highestSeverity(session: Session): LockSeverity | null {
    if (session.locks.length === 0) return null
    return session.locks.reduce<LockSeverity>(
        (highest, lock) => (SEVERITY_RANK[lock.severity] > SEVERITY_RANK[highest] ? lock.severity : highest),
        session.locks[0].severity,
    )
}

interface SessionTableProps {
    sessions: Session[]
}

export function SessionTable({ sessions }: SessionTableProps) {
    if (sessions.length === 0) {
        return <p className="session-table__empty">No active sessions right now.</p>
    }

    return (
        <table className="session-table">
            <thead>
                <tr>
                    <th />
                    <th>PID</th>
                    <th>Duration</th>
                    <th>User / App</th>
                    <th>Query</th>
                    <th>Wait Event</th>
                    <th>Locks</th>
                    <th>Blocked By</th>
                </tr>
            </thead>
            <tbody>
                {sessions.map((session) => (
                    <SessionRow key={session.id} session={session} />
                ))}
            </tbody>
        </table>
    )
}

function SessionRow({ session }: { session: Session }) {
    const severity = highestSeverity(session)
    const status = getSessionStatus(session)

    return (
        <tr className={`state-${status}`}>
            <td className="session-table__state-cell" />
            <td className="mono">{session.id}</td>
            <td className="mono">{formatDuration(session.durationSeconds)}</td>
            <td>
                <div className="session-table__user-app">
                    <span className="session-table__user">{session.user || 'unknown'}</span>
                    <span className="session-table__app">{session.applicationName || '—'}</span>
                </div>
            </td>
            <td className="session-table__query">{session.query}</td>
            <td className={session.waitEventType === 'lock' ? 'session-table__wait session-table__wait--lock' : 'session-table__wait'}>
                {session.waitEventType === 'none' ? '—' : `${session.waitEventType}: ${session.waitEvent}`}
            </td>
            <td>
                <div className="session-table__locks">
                    {severity && <span className={`dot dot--${severity}`} />}
                    <span className="session-table__lock-count">{session.locks.length || '—'}</span>
                </div>
            </td>
            <td className={session.blockedBy.length > 0 ? 'session-table__blocked-by' : 'session-table__blocked-by session-table__blocked-by--none'}>
                {session.blockedBy.length > 0 ? `→ ${session.blockedBy.join(', ')}` : '—'}
            </td>
        </tr>
    )
}