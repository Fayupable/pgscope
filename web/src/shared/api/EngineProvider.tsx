import { useEffect, useState, type ReactNode } from 'react'
import { getConnection } from './connectionClient'
import { EngineContext } from './engineContext'
import type { Engine } from '../types/connection'

/**
 * Fetches which database engine this server is connected to, once, and
 * makes it available to the whole app via context. The engine never
 * changes mid-session (it's fixed by how the backend was started), so a
 * one-time fetch is enough — no polling, no SSE.
 */
export function EngineProvider({ children }: { children: ReactNode }) {
    const [engine, setEngine] = useState<Engine | null>(null)

    useEffect(() => {
        getConnection()
            .then((connection) => setEngine(connection.engine))
            .catch(() => {
                // Leave engine as null on failure — consumers already treat
                // null as "unknown," which is the honest state here; no
                // separate error UI is worth building for a field this
                // low-stakes (only affects which advisory cards render).
            })
    }, [])

    return <EngineContext.Provider value={engine}>{children}</EngineContext.Provider>
}
