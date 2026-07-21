import { describe, expect, it } from 'vitest'
import { findBridges, type GraphEdge } from './edges'

function chainEdges(ids: string[]): GraphEdge[] {
    const edges: GraphEdge[] = []
    for (let i = 0; i < ids.length - 1; i++) {
        edges.push({ source: ids[i], target: ids[i + 1] })
    }
    return edges
}

function starEdges(hub: string, spokes: string[]): GraphEdge[] {
    return spokes.map((spoke) => ({ source: hub, target: spoke }))
}

describe('findBridges', () => {
    it('never flags any edge in a plain chain with no hub anywhere', () => {
        const ids = ['a', 'b', 'c', 'd', 'e', 'f']
        const edges = chainEdges(ids)

        expect(findBridges(ids, edges)).toEqual([])
    })

    it('does not fragment a chain whose connector node also links to an unrelated hub', () => {
        // Regression test for the original bug: node "d" is both an
        // interior chain node (c-d-e-f) and directly linked to a hub
        // ("hub" with 6 spokes). Every interior chain edge must stay
        // non-bridge even though one side of some cuts reaches the hub.
        const hubSpokes = ['s1', 's2', 's3', 's4', 's5', 'd']
        const ids = ['hub', ...hubSpokes.filter((s) => s !== 'd'), 'd', 'c', 'b', 'a']
        const edges: GraphEdge[] = [
            ...starEdges('hub', hubSpokes),
            { source: 'd', target: 'c' },
            { source: 'c', target: 'b' },
            { source: 'b', target: 'a' },
        ]

        const bridges = findBridges(ids, edges)

        expect(bridges).toEqual([])
    })

    it('flags every edge on the sole path connecting two independent hubs', () => {
        // hub1 and hub2 are joined by a single path with no cycle:
        // hub1-a1-connector-b1-hub2. Every edge on that path is a true
        // graph-theoretic bridge, and cutting any one of them still
        // leaves the other three intact — so both sides of any single
        // cut still reach a hub. All four edges should qualify.
        const hub1Spokes = ['a1', 'a2', 'a3', 'a4']
        const hub2Spokes = ['b1', 'b2', 'b3', 'b4']
        const ids = ['hub1', ...hub1Spokes, 'hub2', ...hub2Spokes, 'connector']
        const edges: GraphEdge[] = [
            ...starEdges('hub1', hub1Spokes),
            ...starEdges('hub2', hub2Spokes),
            { source: 'a1', target: 'connector' },
            { source: 'b1', target: 'connector' },
        ]

        const bridges = findBridges(ids, edges)

        expect(bridges).toContainEqual({ source: 'hub1', target: 'a1' })
        expect(bridges).toContainEqual({ source: 'a1', target: 'connector' })
        expect(bridges).toContainEqual({ source: 'b1', target: 'connector' })
        expect(bridges).toContainEqual({ source: 'hub2', target: 'b1' })
        expect(bridges).toHaveLength(4)
    })

    it('does not flag a bridge when only one side has a hub of its own', () => {
        // A hub connected to a single unrelated node — the far side never
        // has its own hub, so this should not be treated as a bridge
        // between two clusters.
        const hubSpokes = ['s1', 's2', 's3']
        const ids = ['hub', ...hubSpokes, 'lonely']
        const edges: GraphEdge[] = [...starEdges('hub', hubSpokes), { source: 'hub', target: 'lonely' }]

        expect(findBridges(ids, edges)).toEqual([])
    })
})
