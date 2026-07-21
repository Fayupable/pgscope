import { useState } from 'react'
import { useSessionStream } from '../../sessions/hooks/useSessionStream'
import { useSnapshotPlayer } from '../../replay/hooks/useSnapshotPlayer'
import { SessionList } from '../../sessions/components/SessionList'
import { SessionGraph } from '../../lock-graph/components/SessionGraph'
import { DbStatsBar } from '../../db-stats/components/DbStatsBar'
import { MonitoringControls } from './MonitoringControls'
import { ReplayControls } from '../../replay/components/ReplayControls'
import { InsightsPanel } from '../../insights/components/InsightsPanel'
import type { Theme } from '../../../shared/hooks/useTheme'
import './Dashboard.css'

type View = 'list' | 'graph'
type Source = 'live' | 'replay' | 'insights'

interface DashboardProps {
    onLogout: () => void
    theme: Theme
    onToggleTheme: () => void
}

export function Dashboard({ onLogout, theme, onToggleTheme }: DashboardProps) {
    const liveSessions = useSessionStream()
    // Owned here (not inside ReplayControls) so a loaded/playing snapshot
    // survives switching away to Live and back — the player would otherwise
    // unmount along with ReplayControls and lose all its state.
    const player = useSnapshotPlayer()

    const [view, setView] = useState<View>('list')
    const [source, setSource] = useState<Source>('live')

    const replaySessions = player.currentSnapshot?.sessions ?? []
    const displayedSessions = source === 'live' ? liveSessions : replaySessions
    const sessionsOverride = source === 'replay' ? replaySessions : undefined

    return (
        <main className="dashboard">
            <header className="dashboard__header">
                <div>
                    <h1>pgscope</h1>
                    <p className="dashboard__subtitle">
                        {source === 'insights'
                            ? 'Advisory suggestions'
                            : `${displayedSessions.length} active session${displayedSessions.length === 1 ? '' : 's'}`}
                    </p>
                </div>
                <div className="dashboard__header-actions">
                    <button
                        className="dashboard__theme-toggle"
                        onClick={onToggleTheme}
                        title={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
                    >
                        {theme === 'dark' ? '☀' : '🌙'}
                    </button>
                    <button className="dashboard__logout" onClick={onLogout}>
                        Log out
                    </button>
                </div>
            </header>

            <div className="control-panel">
                <div className="control-panel__row">
                    <div className="segmented-control">
                        <button className={source === 'live' ? 'active' : ''} onClick={() => setSource('live')}>
                            Live
                        </button>
                        <button className={source === 'replay' ? 'active' : ''} onClick={() => setSource('replay')}>
                            Replay
                        </button>
                        <button className={source === 'insights' ? 'active' : ''} onClick={() => setSource('insights')}>
                            Insights
                        </button>
                    </div>

                    {source !== 'insights' && (
                        <>
                            <div className="control-panel__divider" />

                            <div className="segmented-control">
                                <button className={view === 'list' ? 'active' : ''} onClick={() => setView('list')}>
                                    List
                                </button>
                                <button className={view === 'graph' ? 'active' : ''} onClick={() => setView('graph')}>
                                    Graph
                                </button>
                            </div>
                        </>
                    )}
                </div>

                <div className="control-panel__body">
                    {source === 'live' && <MonitoringControls />}
                    {source === 'replay' && <ReplayControls player={player} />}
                </div>
            </div>

            {source === 'live' && <DbStatsBar />}

            {source === 'insights' ? (
                <InsightsPanel />
            ) : view === 'list' ? (
                <SessionList sessionsOverride={sessionsOverride} />
            ) : (
                <SessionGraph sessionsOverride={sessionsOverride} />
            )}
        </main>
    )
}