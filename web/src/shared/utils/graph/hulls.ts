import { polygonHull, polygonCentroid } from 'd3-polygon'
import type { Position } from './types'

export interface ClusterHull {
    points: Position[]
    colorIndex: number
}

const HULL_PADDING = 36
const HULL_COLOR_COUNT = 5
/** If the thinness ratio (area / perimeter^2) is below this, treat as near-collinear. */
const COLLINEAR_AREA_RATIO = 1e-4

/**
 * Computes a padded convex hull polygon per multi-node cluster, using each
 * cluster's final (packed) node positions. Padding is applied by pushing
 * every hull point outward from the hull's centroid, so the shape visually
 * surrounds the nodes with some margin rather than hugging them tightly.
 * Clusters with only 2 nodes (no true hull, d3-polygon needs 3+ points)
 * fall back to a small rectangle around the two points.
 *
 * Near-collinear 3+ node layouts (common for tiny force-simulated clusters)
 * also use the rectangle fallback: centroid padding on a razor-thin hull
 * produces huge perpendicular overshoot.
 */
export function computeClusterHulls(
    componentLayouts: { ids: string[]; positions: Map<string, Position> }[],
    packedPositions: Map<string, Position>,
): ClusterHull[] {
    return componentLayouts.map((component, index) => {
        const points: [number, number][] = component.ids.map((id) => {
            const pos = packedPositions.get(id) ?? { x: 0, y: 0 }
            return [pos.x, pos.y]
        })

        const hull = polygonHull(points)
        const colorIndex = index % HULL_COLOR_COUNT

        if (!hull || isNearlyCollinear(hull)) {
            return { points: padRectangle(points), colorIndex }
        }

        const centroid = polygonCentroid(hull)
        const padded = hull.map(([x, y]) => padPoint(x, y, centroid[0], centroid[1]))

        return { points: padded.map(([x, y]) => ({ x, y })), colorIndex }
    })
}

function isNearlyCollinear(hull: [number, number][]): boolean {
    if (hull.length < 3) return true
    // Shoelace area
    let area2 = 0
    let peri = 0
    for (let i = 0; i < hull.length; i++) {
        const [x1, y1] = hull[i]
        const [x2, y2] = hull[(i + 1) % hull.length]
        area2 += x1 * y2 - x2 * y1
        peri += Math.hypot(x2 - x1, y2 - y1)
    }
    const area = Math.abs(area2) / 2
    if (peri <= 0) return true
    // Dimensionless thinness; collinear/flat hulls have near-zero area.
    return area / (peri * peri) < COLLINEAR_AREA_RATIO
}

function padPoint(x: number, y: number, centroidX: number, centroidY: number): [number, number] {
    const dx = x - centroidX
    const dy = y - centroidY
    const length = Math.sqrt(dx * dx + dy * dy) || 1
    return [x + (dx / length) * HULL_PADDING, y + (dy / length) * HULL_PADDING]
}

function padRectangle(points: [number, number][]): Position[] {
    const xs = points.map((p) => p[0])
    const ys = points.map((p) => p[1])
    const minX = Math.min(...xs) - HULL_PADDING
    const maxX = Math.max(...xs) + HULL_PADDING
    const minY = Math.min(...ys) - HULL_PADDING
    const maxY = Math.max(...ys) + HULL_PADDING

    return [
        { x: minX, y: minY },
        { x: maxX, y: minY },
        { x: maxX, y: maxY },
        { x: minX, y: maxY },
    ]
}
