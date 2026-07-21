export type SessionState = 'active' | 'idle' | 'idle_in_transaction' | 'disabled'

export type WaitEventType = 'none' | 'lock' | 'io' | 'cpu' | 'client' | 'ipc' | 'timeout'

export type LockSeverity = 'unknown' | 'shared_read' | 'shared_write' | 'intent' | 'exclusive'

export type QueryOperation =
    | 'SELECT'
    | 'INSERT'
    | 'UPDATE'
    | 'DELETE'
    | 'MERGE'
    | 'DDL'
    | 'VACUUM'
    | 'OTHER'
    | 'UNKNOWN'

export interface LockedObject {
    nativeMode: string
    severity: LockSeverity
    resource?: string
    granted: boolean
}

export interface Session {
    id: string
    user: string
    applicationName: string
    clientAddress: string
    state: SessionState
    waitEventType: WaitEventType
    waitEvent: string
    query: string
    operation: QueryOperation
    queryStarted: string
    durationSeconds: number
    blockedBy: string[]
    locks: LockedObject[]
}

export interface DatabaseActivityStats {
    commitsPerSecond: number
    rollbacksPerSecond: number
    cacheHitRatio: number
    measuredAt: string
}