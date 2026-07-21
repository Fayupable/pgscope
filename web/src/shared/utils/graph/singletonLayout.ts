import { forceSimulation, forceManyBody, forceCollide, type SimulationNodeDatum } from 'd3-force'
import { boundingBoxOf } from './geometry'
import { CANVAS_WIDTH } from './packing'
import type { Position } from './types'

interface ForceNode extends SimulationNodeDatum {
    id: string
}

const SINGLETON_COLLIDE_RADIUS = 32
const SINGLETON_TICKS = 220
const SINGLETON_CHARGE = -70
const AREA_HEIGHT_PER_NODE = 2600

/**
 * Scatters fully isolated (zero-edge) sessions organically across a wide
 * 2D area instead of confining them to a tight grid/strip — each node is
 * seeded at a fully random position within a generous rectangle (sized to
 * the node count), then relaxed with repulsion + collision only (no links,
 * since there's nothing to connect). The random seed plus a wide available
 * area is what produces a natural, spread-out "scattered" look rather than
 * a mechanical row/column pattern or a dense confined strip.
 */
export function layoutSingletons(ids: string[], startY: number): Map<string, Position> {
    if (ids.length === 0) return new Map()

    const areaHeight = Math.max(200, Math.sqrt(ids.length) * AREA_HEIGHT_PER_NODE / CANVAS_WIDTH)

    const nodes: ForceNode[] = ids.map((id) => ({
        id,
        x: Math.random() * CANVAS_WIDTH,
        y: Math.random() * areaHeight,
    }))

    const simulation = forceSimulation(nodes)
        .force('charge', forceManyBody().strength(SINGLETON_CHARGE))
        .force('collide', forceCollide(SINGLETON_COLLIDE_RADIUS))
        .stop()

    simulation.alpha(1)
    simulation.tick(SINGLETON_TICKS)

    const rawPositions = new Map(nodes.map((n) => [n.id, { x: n.x ?? 0, y: n.y ?? 0 }]))
    const box = boundingBoxOf(rawPositions)
    const offsetY = startY - box.minY

    return new Map(nodes.map((n) => [n.id, { x: n.x ?? 0, y: (n.y ?? 0) + offsetY }]))
}