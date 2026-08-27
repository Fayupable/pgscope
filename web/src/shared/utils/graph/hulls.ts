import { polygonHull } from 'd3-polygon'
import type { Position } from './types'

export interface ClusterHull {
    points: Position[]
    colorIndex: number
}

// Half of the 44px session node rendered in SessionGraph / ClusterHulls.
// Hulls are computed from top-left anchors; ClusterHulls shifts by this
// amount so the path is centered on the visible circles.
const NODE_HALF_SIZE = 22
// Extra air gap outside each node circle so the hull never appears to
// sit on (or clip through) the node boundary.
const HULL_MARGIN = 20
/** Clearance from each node anchor to the hull — node radius + visible margin. */
const HULL_PADDING = NODE_HALF_SIZE + HULL_MARGIN
const HULL_COLOR_COUNT = 5

/**
 * Computes a padded convex hull polygon per multi-node cluster, using each
 * cluster's final (packed) node positions.
 *
 * Padding is the convex hull of axis-aligned squares of half-side
 * HULL_PADDING around every node anchor. That is equivalent to a Minkowski
 * sum of the point set with a square, and it:
 *   - guarantees at least HULL_PADDING clearance from every node (so the
 *     rendered 44px circles never sit on the hull edge)
 *   - naturally becomes a padded bounding rectangle for near-collinear
 *     layouts, avoiding the huge perpendicular overshoot that centroid
 *     radial expansion produced on razor-thin hulls
 * Clusters with only one distinct point still get a small square.
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

        const colorIndex = index % HULL_COLOR_COUNT
        return { points: padAroundNodes(points), colorIndex }
    })
}

/**
 * Expand each node to a square of half-side HULL_PADDING and take the
 * convex hull of all square corners.
 */
function padAroundNodes(points: [number, number][]): Position[] {
    if (points.length === 0) return []

    const inflated: [number, number][] = []
    for (const [x, y] of points) {
        inflated.push(
            [x - HULL_PADDING, y - HULL_PADDING],
            [x + HULL_PADDING, y - HULL_PADDING],
            [x + HULL_PADDING, y + HULL_PADDING],
            [x - HULL_PADDING, y + HULL_PADDING],
        )
    }

    const hull = polygonHull(inflated)
    if (!hull) {
        // Degenerate (should not happen with 4+ inflated corners); fall back
        // to the AABB of the inflated set.
        return padRectangle(points)
    }
    return hull.map(([x, y]) => ({ x, y }))
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

/** Exported for unit tests. */
export const _test = { HULL_PADDING, NODE_HALF_SIZE, HULL_MARGIN, padAroundNodes }