import { useEffect, useMemo, useRef, useState } from 'react'
import { ReactFlow, Background, Controls, type Node, type Edge, type NodeMouseHandler, type EdgeMouseHandler } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useSessionStream } from '../../sessions/hooks/useSessionStream'
import { getSessionStatus, STATUS_COLOR_VAR } from '../../../shared/utils/sessionStatus'
import { computeClusteredLayout, type ClusterHull } from '../../../shared/utils/graph'
import type { Session } from '../../../shared/types/session'
import { SessionDetailPanel } from './SessionDetailPanel'
import { EdgeDetailPanel } from './EdgeDetailPanel'
import { ClusterHulls } from './ClusterHulls'
import { GraphLegend } from './GraphLegend'
import './SessionGraph.css'

function buildFlowElements(sessions: Session[]): { nodes: Node[]; edges: Edge[]; clusterHulls: ClusterHull[] } {
    const { positions, clusterEdges, bridgeEdges, clusterHulls } = computeClusteredLayout(sessions)

    const nodes: Node[] = sessions.map((session) => {
        const position = positions.get(session.id) ?? { x: 0, y: 0 }

        return {
            id: session.id,
            position,
            data: { label: session.id },
            style: {
                background: 'var(--bg-elevated)',
                border: `2px solid ${STATUS_COLOR_VAR[getSessionStatus(session)]}`,
                borderRadius: '50%',
                width: 44,
                height: 44,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                cursor: 'pointer',
            },
        }
    })

    const clusterFlowEdges: Edge[] = clusterEdges.map((e) => ({
        id: `${e.source}->${e.target}`,
        source: e.source,
        target: e.target,
        animated: true,
        style: { stroke: 'var(--lock-exclusive)' },
    }))

    const bridgeFlowEdges: Edge[] = bridgeEdges.map((e) => ({
        id: `bridge-${e.source}->${e.target}`,
        source: e.source,
        target: e.target,
        animated: false,
        style: { stroke: 'var(--text-muted)', strokeWidth: 1 },
    }))

    return { nodes, edges: [...clusterFlowEdges, ...bridgeFlowEdges], clusterHulls }
}

interface SessionGraphProps {
    sessionsOverride?: Session[]
}

export function SessionGraph({ sessionsOverride }: SessionGraphProps) {
    const liveSessions = useSessionStream()
    const sessions = sessionsOverride ?? liveSessions

    const { nodes, edges, clusterHulls } = useMemo(() => buildFlowElements(sessions), [sessions])
    const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null)
    const [selectedEdge, setSelectedEdge] = useState<{ blockerId: string; blockedId: string } | null>(null)
    const [panelPosition, setPanelPosition] = useState({ x: 0, y: 0 })
    const containerRef = useRef<HTMLDivElement>(null)

    const selectedSession = sessions.find((s) => s.id === selectedSessionId) ?? null
    const edgeBlocker = sessions.find((s) => s.id === selectedEdge?.blockerId) ?? null
    const edgeBlocked = sessions.find((s) => s.id === selectedEdge?.blockedId) ?? null

    useEffect(() => {
        if (!selectedSessionId && !selectedEdge) return

        function handleKeyDown(event: KeyboardEvent) {
            if (event.key === 'Escape') {
                setSelectedSessionId(null)
                setSelectedEdge(null)
            }
        }

        window.addEventListener('keydown', handleKeyDown)
        return () => window.removeEventListener('keydown', handleKeyDown)
    }, [selectedSessionId, selectedEdge])

    function updatePanelPosition(clientX: number, clientY: number) {
        const containerRect = containerRef.current?.getBoundingClientRect()
        if (containerRect) {
            setPanelPosition({
                x: clientX - containerRect.left + 16,
                y: clientY - containerRect.top,
            })
        }
    }

    const handleNodeClick: NodeMouseHandler = (event, node) => {
        updatePanelPosition(event.clientX, event.clientY)
        setSelectedEdge(null)
        setSelectedSessionId(node.id)
    }

    const handleEdgeClick: EdgeMouseHandler = (event, edge) => {
        updatePanelPosition(event.clientX, event.clientY)
        setSelectedSessionId(null)
        setSelectedEdge({ blockerId: edge.source, blockedId: edge.target })
    }

    if (sessions.length === 0) {
        return <p className="session-graph__empty">No active sessions right now.</p>
    }

    return (
        <div className="session-graph-wrapper">
            <div className="session-graph" ref={containerRef}>
                <ReactFlow nodes={nodes} edges={edges} onNodeClick={handleNodeClick} onEdgeClick={handleEdgeClick} fitView>
                    <Background color="var(--border-subtle)" gap={24} />
                    <ClusterHulls hulls={clusterHulls} />
                    <Controls showInteractive={false} />
                </ReactFlow>

                {selectedSession && (
                    <SessionDetailPanel
                        session={selectedSession}
                        position={panelPosition}
                        onClose={() => setSelectedSessionId(null)}
                    />
                )}

                {edgeBlocker && edgeBlocked && (
                    <EdgeDetailPanel
                        blocker={edgeBlocker}
                        blocked={edgeBlocked}
                        position={panelPosition}
                        onClose={() => setSelectedEdge(null)}
                    />
                )}
            </div>

            <GraphLegend />
        </div>
    )
}