import { useSessionStream } from '../hooks/useSessionStream'
import { SessionTable } from './SessionTable'
import type { Session } from '../../../shared/types/session'

interface SessionListProps {
    sessionsOverride?: Session[]
}

export function SessionList({ sessionsOverride }: SessionListProps) {
    const liveSessions = useSessionStream()
    const sessions = sessionsOverride ?? liveSessions

    return <SessionTable sessions={sessions} />
}