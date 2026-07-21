import { forceSimulation, forceLink, forceManyBody, forceCollide, type SimulationNodeDatum } from 'd3-force'
import type { GraphEdge } from './edges'
import type { Position } from './types'

interface ForceNode extends SimulationNodeDatum {
    id: string
}

const SIMULATION_TICKS = 300
const LINK_DISTANCE = 90
const LINK_STRENGTH = 0.5
const CHARGE_STRENGTH = -180
const COLLIDE_RADIUS = 28

/**
 * Lays out a single connected component (2+ nodes) with d3-force. The hub
 * (highest-degree node) is pinned at the component's center; every other
 * node is explicitly seeded on a jittered circle around it before the
 * simulation starts, so the simulation has a genuine 2D configuration to
 * relax from instead of collapsing from a near-coincident start.
 */
export function layoutComponent(
    nodeIds: string[],
    edges: GraphEdge[],
    centerX: number,
    centerY: number,
): Map<string, Position> {
    if (nodeIds.length === 1) {
        return new Map([[nodeIds[0], { x: centerX, y: centerY }]])
    }

    const hubId = findHub(nodeIds, edges)
    const nodes: ForceNode[] = nodeIds.map((id) => ({ id }))

    seedInitialPositions(nodes, hubId, centerX, centerY)

    const links = edges.map((e) => ({ source: e.source, target: e.target }))

    const simulation = forceSimulation(nodes)
        .force(
            'link',
            forceLink(links).id((d) => (d as ForceNode).id).distance(LINK_DISTANCE).strength(LINK_STRENGTH),
        )
        .force('charge', forceManyBody().strength(CHARGE_STRENGTH))
        .force('collide', forceCollide(COLLIDE_RADIUS))
        .stop()

    simulation.alpha(1)
    simulation.tick(SIMULATION_TICKS)

    return new Map(nodes.map((n) => [n.id, { x: n.x ?? centerX, y: n.y ?? centerY }]))
}

function seedInitialPositions(nodes: ForceNode[], hubId: string, centerX: number, centerY: number): void {
    const others = nodes.filter((n) => n.id !== hubId)
    const angleStep = (2 * Math.PI) / Math.max(others.length, 1)

    for (const node of nodes) {
        if (node.id === hubId) {
            node.x = centerX
            node.y = centerY
            node.fx = centerX
            node.fy = centerY
        }
    }

    others.forEach((node, i) => {
        const jitter = (Math.random() - 0.5) * 0.5
        const angle = i * angleStep + jitter
        node.x = centerX + Math.cos(angle) * LINK_DISTANCE
        node.y = centerY + Math.sin(angle) * LINK_DISTANCE
    })
}

function findHub(nodeIds: string[], edges: GraphEdge[]): string {
    const degree = new Map<string, number>(nodeIds.map((id) => [id, 0]))

    for (const edge of edges) {
        degree.set(edge.source, (degree.get(edge.source) ?? 0) + 1)
        degree.set(edge.target, (degree.get(edge.target) ?? 0) + 1)
    }

    return [...degree.entries()].sort((a, b) => b[1] - a[1])[0][0]
}