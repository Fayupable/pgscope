import { useState } from 'react'
import { useInsights } from '../hooks/useInsights'
import { TopQueriesTable } from './TopQueriesTable'
import { IndexCandidatesTable } from './IndexCandidatesTable'
import { DuplicateIndexesTable } from './DuplicateIndexesTable'
import { UnusedIndexesTable } from './UnusedIndexesTable'
import { FunctionCostsTable } from './FunctionCostsTable'
import { PaginationWarningsTable } from './PaginationWarningsTable'
import { HealthPanel } from './HealthPanel'
import './InsightsPanel.css'

type Tab = 'queries' | 'candidates' | 'duplicates' | 'unused' | 'functions' | 'pagination' | 'health'

export function InsightsPanel() {
    const { insights, loading, error, refresh } = useInsights()
    const [tab, setTab] = useState<Tab>('queries')

    return (
        <div className="insights-panel">
            <div className="insights-panel__toolbar">
                <div className="segmented-control">
                    <button className={tab === 'queries' ? 'active' : ''} onClick={() => setTab('queries')}>
                        Top Queries
                    </button>
                    <button className={tab === 'candidates' ? 'active' : ''} onClick={() => setTab('candidates')}>
                        Index Candidates
                    </button>
                    <button className={tab === 'duplicates' ? 'active' : ''} onClick={() => setTab('duplicates')}>
                        Duplicate Indexes
                    </button>
                    <button className={tab === 'unused' ? 'active' : ''} onClick={() => setTab('unused')}>
                        Unused Indexes
                    </button>
                    <button className={tab === 'functions' ? 'active' : ''} onClick={() => setTab('functions')}>
                        Functions & Triggers
                    </button>
                    <button className={tab === 'pagination' ? 'active' : ''} onClick={() => setTab('pagination')}>
                        Pagination Warnings
                    </button>
                    <button className={tab === 'health' ? 'active' : ''} onClick={() => setTab('health')}>
                        Health
                    </button>
                </div>

                <button onClick={refresh} disabled={loading}>
                    {loading ? 'Refreshing…' : 'Refresh'}
                </button>

                {error && <p className="insights-panel__error">{error}</p>}
            </div>

            {tab === 'queries' && (
                <section className="insights-panel__section">
                    <TopQueriesTable queries={insights?.topQueries ?? []} />
                </section>
            )}

            {tab === 'candidates' && (
                <section className="insights-panel__section">
                    <IndexCandidatesTable candidates={insights?.indexCandidates ?? []} />
                </section>
            )}

            {tab === 'duplicates' && (
                <section className="insights-panel__section">
                    <DuplicateIndexesTable duplicates={insights?.duplicateIndexes ?? []} />
                </section>
            )}

            {tab === 'unused' && (
                <section className="insights-panel__section">
                    <UnusedIndexesTable indexes={insights?.unusedIndexes ?? []} />
                </section>
            )}

            {tab === 'functions' && (
                <section className="insights-panel__section">
                    <FunctionCostsTable
                        functions={insights?.functionCosts ?? []}
                        trackFunctionsEnabled={insights?.trackFunctionsEnabled ?? false}
                    />
                </section>
            )}

            {tab === 'pagination' && (
                <section className="insights-panel__section">
                    <PaginationWarningsTable
                        warnings={insights?.paginationWarnings ?? []}
                        nestedStatementsTracked={insights?.nestedStatementsTracked ?? false}
                    />
                </section>
            )}

            {tab === 'health' && insights && (
                <section className="insights-panel__section">
                    <HealthPanel
                        databaseSize={insights.databaseSize}
                        connectionSaturation={insights.connectionSaturation}
                        sequenceOverflowRisks={insights.sequenceOverflowRisks}
                        invalidIndexes={insights.invalidIndexes}
                        unvalidatedConstraints={insights.unvalidatedConstraints}
                        vacuumHealthWarnings={insights.vacuumHealthWarnings}
                        idleInTransactionWarnings={insights.idleInTransactionWarnings}
                        checkpointHealth={insights.checkpointHealth}
                        replicationLagWarnings={insights.replicationLagWarnings}
                        physicalIOEnabled={insights.physicalIOEnabled}
                        physicalIOHotspots={insights.physicalIOHotspots}
                    />
                </section>
            )}
        </div>
    )
}