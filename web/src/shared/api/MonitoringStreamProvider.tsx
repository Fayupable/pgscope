import { useEffect, useState, type ReactNode } from 'react'
import { connectToStream } from './sseClient'
import { MonitoringStreamContext } from './monitoringStreamContext'
import type { Session, DatabaseActivityStats } from '../types/session'

const SESSIONS_STREAM_URL = '/api/v1/sessions/stream'

const EMPTY_STATS: DatabaseActivityStats = {
    commitsPerSecond: 0,
    rollbacksPerSecond: 0,
    cacheHitRatio: 0,
    tempBytesPerSecond: 0,
    measuredAt: '',
}

/**
 * Opens a single SSE connection for the whole app and fans both event types
 * out to consumers via context, instead of each hook opening its own
 * EventSource against the same endpoint.
 */
export function MonitoringStreamProvider({ children }: { children: ReactNode }) {
    const [sessions, setSessions] = useState<Session[]>([])
    const [dbStats, setDbStats] = useState<DatabaseActivityStats>(EMPTY_STATS)

    useEffect(() => {
        const disconnect = connectToStream(SESSIONS_STREAM_URL, {
            sessions: (payload) => setSessions(payload as Session[]),
            db_stats: (payload) => setDbStats(payload as DatabaseActivityStats),
        })

        return disconnect
    }, [])

    return (
        <MonitoringStreamContext.Provider value={{ sessions, dbStats }}>
            {children}
        </MonitoringStreamContext.Provider>
    )
}