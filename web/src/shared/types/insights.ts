export type Confidence = 'strong' | 'weak' | 'insufficient_data'

export interface SlowQuery {
    query: string
    calls: number
    totalExecMs: number
    meanExecMs: number
}

export interface IndexCandidate {
    table: string
    seqScanCount: number
    idxScanCount: number
    suspectedColumns: string[]
    existingIndexCount: number
    writesPerSecond: number
    confidence: Confidence
    rationale: string
    selectivity?: number
}

export interface DuplicateIndex {
    table: string
    redundantIndex: string
    coveringIndex: string
    redundantColumns: string[]
    coveringColumns: string[]
    explanation: string
}

export interface UnusedIndex {
    table: string
    index: string
    sizeBytes: number
    indexScans: number
    explanation: string
}

export interface FunctionCost {
    function: string
    calls: number
    selfTimeMs: number
    isTrigger: boolean
    triggerTables?: string[]
    explanation: string
}

export interface PaginationWarning {
    query: string
    warning: string
}

export interface Insights {
    topQueries: SlowQuery[]
    indexCandidates: IndexCandidate[]
    duplicateIndexes: DuplicateIndex[]
    unusedIndexes: UnusedIndex[]
    functionCosts: FunctionCost[]
    trackFunctionsEnabled: boolean
    trackFunctionsSetting: string
    paginationWarnings: PaginationWarning[]
    nestedStatementsTracked: boolean
    statementsTrackSetting: string
    databaseSize: DatabaseSizeInfo
    connectionSaturation: ConnectionSaturation
    sequenceOverflowRisks: SequenceOverflowRisk[]
    invalidIndexes: InvalidIndex[]
    unvalidatedConstraints: UnvalidatedConstraint[]
    vacuumHealthWarnings: VacuumHealthWarning[]
    idleInTransactionWarnings: IdleInTransactionWarning[]
    checkpointHealth: CheckpointHealth
    replicationLagWarnings: ReplicationLagWarning[]
    physicalIOEnabled: boolean
    physicalIOHotspots: PhysicalIOHotspot[]
    preparedTransactionWarnings: PreparedTransactionWarning[]
    replicationSlotWarnings: ReplicationSlotWarning[]
    longRunningQueryWarnings: LongRunningQueryWarning[]
    unloggedTables: UnloggedTable[]
    lockWaitWarnings: LockWaitWarning[]
}

export interface TableSize {
    table: string
    totalBytes: number
}

export interface DatabaseSizeInfo {
    totalBytes: number
    largestTables: TableSize[]
}

export interface ConnectionSaturation {
    activeConnections: number
    maxConnections: number
    usagePercent: number
    warning?: string
}

export interface SequenceOverflowRisk {
    sequence: string
    currentValue: number
    maxValue: number
    usagePercent: number
    explanation: string
}

export interface InvalidIndex {
    table: string
    index: string
    explanation: string
}

export interface UnvalidatedConstraint {
    table: string
    constraint: string
    explanation: string
}

export interface VacuumHealthWarning {
    table: string
    deadTupleRatio: number
    lastAutovacuum?: string
    explanation: string
}

export interface IdleInTransactionWarning {
    pid: number
    user: string
    applicationName: string
    idleSeconds: number
    explanation: string
}

export interface CheckpointHealth {
    scheduledCheckpoints: number
    requestedCheckpoints: number
    requestedRatio: number
    warning?: string
}

export interface ReplicationLagWarning {
    applicationName: string
    clientAddr: string
    lagBytes: number
    explanation: string
}

export interface PhysicalIOHotspot {
    query: string
    calls: number
    execReads: number
    execWrites: number
    userTimeMs: number
    systemTimeMs: number
    explanation: string
}

export interface PreparedTransactionWarning {
    gid: string
    database: string
    owner: string
    ageSeconds: number
    explanation: string
}

export interface ReplicationSlotWarning {
    slotName: string
    active: boolean
    retainedBytes: number
    explanation: string
}

export interface LongRunningQueryWarning {
    pid: number
    user: string
    applicationName: string
    query: string
    runningSeconds: number
    explanation: string
}

export interface UnloggedTable {
    table: string
    explanation: string
}

// LockWaitWarning is MySQL-only for now — Postgres responses never include
// this field. See domain.LockWaitSession on the backend for why.
export interface LockWaitWarning {
    waitingPid: number
    waitingQuery: string
    blockingPid: number
    blockingQuery: string
    lockedTable: string
    waitAgeSeconds: number
    explanation: string
}