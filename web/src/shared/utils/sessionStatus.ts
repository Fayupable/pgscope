import type { Session } from '../types/session'

export type SessionStatus = 'active' | 'longrunning' | 'blocked' | 'idletx' | 'idle'

const LONG_RUNNING_THRESHOLD_SECONDS = 5

export function getSessionStatus(session: Session): SessionStatus {
    if (session.blockedBy.length > 0) return 'blocked'
    if (session.state === 'idle_in_transaction') return 'idletx'
    if (session.state === 'idle') return 'idle'
    if (session.state === 'active' && session.durationSeconds > LONG_RUNNING_THRESHOLD_SECONDS) return 'longrunning'
    return 'active'
}

export const STATUS_COLOR_VAR: Record<SessionStatus, string> = {
    active: 'var(--status-success)',
    longrunning: 'var(--status-warning)',
    blocked: 'var(--status-danger)',
    idletx: 'var(--status-warning)',
    idle: 'var(--status-neutral)',
}