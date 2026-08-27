import { describe, expect, it } from 'vitest'
import { computeClusterHulls, _test } from './hulls'

const { HULL_PADDING, NODE_HALF_SIZE, padAroundNodes } = _test

function minDistancePointToSegment(
    px: number,
    py: number,
    ax: number,
    ay: number,
    bx: number,
    by: number,
): number {
    const abx = bx - ax
    const aby = by - ay
    const apx = px - ax
    const apy = py - ay
    const abLen2 = abx * abx + aby * aby
    if (abLen2 === 0) return Math.hypot(apx, apy)
    let t = (apx * abx + apy * aby) / abLen2
    t = Math.max(0, Math.min(1, t))
    const qx = ax + t * abx
    const qy = ay + t * aby
    return Math.hypot(px - qx, py - qy)
}

/** Minimum distance from a point to the boundary of a closed polygon. */
function minDistanceToPolygon(px: number, py: number, poly: { x: number; y: number }[]): number {
    let min = Infinity
    for (let i = 0; i < poly.length; i++) {
        const a = poly[i]
        const b = poly[(i + 1) % poly.length]
        min = Math.min(min, minDistancePointToSegment(px, py, a.x, a.y, b.x, b.y))
    }
    return min
}

function layoutOf(coords: Record<string, [number, number]>) {
    const ids = Object.keys(coords)
    const positions = new Map(ids.map((id) => [id, { x: coords[id][0], y: coords[id][1] }]))
    return { ids, positions }
}

describe('computeClusterHulls', () => {
    it('keeps at least HULL_PADDING clearance from every node on a 3-node line', () => {
        // Near-collinear 3-node cluster (the overshoot case from #2)
        const component = layoutOf({
            a: [0, 0],
            b: [100, 2],
            c: [200, -1],
        })
        const hulls = computeClusterHulls([component], component.positions)
        expect(hulls).toHaveLength(1)
        const poly = hulls[0].points

        for (const id of component.ids) {
            const p = component.positions.get(id)!
            const dist = minDistanceToPolygon(p.x, p.y, poly)
            expect(dist).toBeGreaterThanOrEqual(HULL_PADDING - 1e-6)
        }
    })

    it('keeps clearance on a 4-node triangular cluster (edge nodes included)', () => {
        // Three corners + one node near an edge (reviewer case: node on boundary)
        const component = layoutOf({
            a: [0, 0],
            b: [200, 0],
            c: [80, 150],
            d: [100, 5], // sits almost on the ab edge
        })
        const hulls = computeClusterHulls([component], component.positions)
        const poly = hulls[0].points

        for (const id of component.ids) {
            const p = component.positions.get(id)!
            const dist = minDistanceToPolygon(p.x, p.y, poly)
            expect(dist).toBeGreaterThanOrEqual(HULL_PADDING - 1e-6)
        }
        // Visible gap outside the node circle
        expect(HULL_PADDING - NODE_HALF_SIZE).toBeGreaterThanOrEqual(16)
    })

    it('does not produce huge perpendicular overshoot on a collinear triple', () => {
        const poly = padAroundNodes([
            [0, 0],
            [100, 0],
            [200, 0],
        ])
        const ys = poly.map((p) => p.y)
        const height = Math.max(...ys) - Math.min(...ys)
        // Square inflation → height exactly 2 * HULL_PADDING (no centroid blow-up)
        expect(height).toBeCloseTo(2 * HULL_PADDING, 6)
        // Span along x should be 200 + 2 * padding
        const xs = poly.map((p) => p.x)
        expect(Math.max(...xs) - Math.min(...xs)).toBeCloseTo(200 + 2 * HULL_PADDING, 6)
    })

    it('assigns rotating color indices per component', () => {
        const a = layoutOf({ a: [0, 0], b: [10, 0] })
        const b = layoutOf({ c: [50, 50], d: [60, 50] })
        const packed = new Map([...a.positions, ...b.positions])
        const hulls = computeClusterHulls([a, b], packed)
        expect(hulls.map((h) => h.colorIndex)).toEqual([0, 1])
    })
})