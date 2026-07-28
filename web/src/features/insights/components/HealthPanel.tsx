import type {
    CheckpointHealth,
    ConnectionSaturation,
    DatabaseSizeInfo,
    IdleInTransactionWarning,
    InvalidIndex,
    LongRunningQueryWarning,
    PhysicalIOHotspot,
    PreparedTransactionWarning,
    ReplicationLagWarning,
    ReplicationSlotWarning,
    SequenceOverflowRisk,
    UnloggedTable,
    UnvalidatedConstraint,
    VacuumHealthWarning,
} from '../../../shared/types/insights'
import { DatabaseSizeCard } from './DatabaseSizeCard'
import { DatabaseSizeTable } from './DatabaseSizeTable'
import { ConnectionSaturationCard } from './ConnectionSaturationCard'
import { SequenceOverflowTable } from './SequenceOverflowTable'
import { InvalidObjectsCard } from './InvalidObjectsCard'
import { VacuumHealthTable } from './VacuumHealthTable'
import { IdleInTransactionTable } from './IdleInTransactionTable'
import { CheckpointHealthCard } from './CheckpointHealthCard'
import { ReplicationLagTable } from './ReplicationLagTable'
import { PhysicalIOHotspotsTable } from './PhysicalIOHotspotsTable'
import { PreparedTransactionsCard } from './PreparedTransactionsCard'
import { ReplicationSlotsCard } from './ReplicationSlotsCard'
import { LongRunningQueriesCard } from './LongRunningQueriesCard'
import { UnloggedTablesCard } from './UnloggedTablesCard'

export function HealthPanel({
    databaseSize,
    connectionSaturation,
    sequenceOverflowRisks,
    invalidIndexes,
    unvalidatedConstraints,
    vacuumHealthWarnings,
    idleInTransactionWarnings,
    checkpointHealth,
    replicationLagWarnings,
    physicalIOEnabled,
    physicalIOHotspots,
    preparedTransactionWarnings,
    replicationSlotWarnings,
    longRunningQueryWarnings,
    unloggedTables,
}: {
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
}) {
    return (
        <div className="health-panel">
            <div className="health-panel__row">
                <ConnectionSaturationCard saturation={connectionSaturation} />
                <DatabaseSizeCard totalBytes={databaseSize.totalBytes} />
                <SequenceOverflowTable risks={sequenceOverflowRisks} />
                <InvalidObjectsCard invalidIndexes={invalidIndexes} unvalidatedConstraints={unvalidatedConstraints} />
                <CheckpointHealthCard health={checkpointHealth} />
                <PreparedTransactionsCard warnings={preparedTransactionWarnings} />
                <ReplicationSlotsCard warnings={replicationSlotWarnings} />
                <LongRunningQueriesCard warnings={longRunningQueryWarnings} />
                <UnloggedTablesCard tables={unloggedTables} />
            </div>

            <div className="health-panel__section-title">Vacuum health</div>
            <VacuumHealthTable warnings={vacuumHealthWarnings} />

            <div className="health-panel__section-title">Idle in transaction</div>
            <IdleInTransactionTable warnings={idleInTransactionWarnings} />

            <div className="health-panel__section-title">Replication lag</div>
            <ReplicationLagTable warnings={replicationLagWarnings} />

            <div className="health-panel__section-title">Physical I/O hotspots</div>
            <PhysicalIOHotspotsTable hotspots={physicalIOHotspots} physicalIOEnabled={physicalIOEnabled} />

            <div className="health-panel__section-title">Largest tables</div>
            <DatabaseSizeTable tables={databaseSize.largestTables} />
        </div>
    )
}
