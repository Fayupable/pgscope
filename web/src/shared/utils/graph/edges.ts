import type { Session } from '../../types/session'

export interface GraphEdge {
    source: string
    target: string
}

export function buildEdges(sessions: Session[]): GraphEdge[] {
    const knownIds = new Set(sessions.map((s) => s.id))
    const seen = new Set<string>()
    const edges: GraphEdge[] = []

    for (const session of sessions) {
        for (const blockerId of session.blockedBy) {
            if (!knownIds.has(blockerId)) continue

            const key = edgeKey({ source: blockerId, target: session.id })
            if (seen.has(key)) continue
            seen.add(key)

            edges.push({ source: blockerId, target: session.id })
        }
    }

    return edges
}

const HUB_DEGREE_THRESHOLD = 3

/**
 * Finds bridge edges — edges whose removal splits the graph into two
 * pieces that are BOTH real multi-node groups (size >= 2 on each side),
 * AND BOTH sides contain a genuine hub (a node with 3+ connections) of
 * their own. Requiring a hub on both sides (not just one) is what keeps
 * a chain node with a single extra cross-link (e.g. a node that's both
 * part of a sequential chain AND directly connected to an unrelated
 * hub) from having every interior chain edge misclassified as a bridge —
 * with an OR rule, cutting anywhere in the chain leaves the hub-bearing
 * side "hub-containing" even though the cut itself has nothing to do
 * with that hub. A plain sequential chain (A-B-C-D-E-F, no hub anywhere)
 * still never produces a bridge, since neither side ever has a hub. A
 * genuine link between two separate hub clusters still correctly
 * qualifies, since both sides have their own hub.
 */
export function findBridges(nodeIds: string[], edges: GraphEdge[]): GraphEdge[] {
    return edges.filter((candidate) => {
        const remaining = edges.filter((e) => e !== candidate)
        const sourceComponent = componentContaining(candidate.source, nodeIds, remaining)

        if (sourceComponent.has(candidate.target)) {
            return false
        }

        const targetComponent = componentContaining(candidate.target, nodeIds, remaining)

        if (sourceComponent.size < 2 || targetComponent.size < 2) {
            return false
        }

        return hasHub(sourceComponent, remaining) && hasHub(targetComponent, remaining)
    })
}

function hasHub(component: Set<string>, edges: GraphEdge[]): boolean {
    const degree = new Map<string, number>()

    for (const edge of edges) {
        if (!component.has(edge.source) || !component.has(edge.target)) continue
        degree.set(edge.source, (degree.get(edge.source) ?? 0) + 1)
        degree.set(edge.target, (degree.get(edge.target) ?? 0) + 1)
    }

    for (const count of degree.values()) {
        if (count >= HUB_DEGREE_THRESHOLD) return true
    }

    return false
}

export function findConnectedComponents(nodeIds: string[], edges: GraphEdge[]): string[][] {
    const adjacency = buildAdjacency(nodeIds, edges)
    const visited = new Set<string>()
    const components: string[][] = []

    for (const nodeId of nodeIds) {
        if (visited.has(nodeId)) continue

        const component: string[] = []
        const queue = [nodeId]
        visited.add(nodeId)

        while (queue.length > 0) {
            const current = queue.shift()!
            component.push(current)

            for (const neighbor of adjacency.get(current) ?? []) {
                if (!visited.has(neighbor)) {
                    visited.add(neighbor)
                    queue.push(neighbor)
                }
            }
        }

        components.push(component)
    }

    return components
}

export function edgesWithin(nodeIds: string[], edges: GraphEdge[]): GraphEdge[] {
    const idSet = new Set(nodeIds)
    return edges.filter((e) => idSet.has(e.source) && idSet.has(e.target))
}

export function edgeKey(edge: GraphEdge): string {
    return [edge.source, edge.target].sort().join('::')
}

export function componentContaining(startId: string, nodeIds: string[], edges: GraphEdge[]): Set<string> {
    const adjacency = buildAdjacency(nodeIds, edges)
    const visited = new Set<string>([startId])
    const queue = [startId]

    while (queue.length > 0) {
        const current = queue.shift()!
        for (const neighbor of adjacency.get(current) ?? []) {
            if (!visited.has(neighbor)) {
                visited.add(neighbor)
                queue.push(neighbor)
            }
        }
    }

    return visited
}

export function buildAdjacency(nodeIds: string[], edges: GraphEdge[]): Map<string, string[]> {
    const adjacency = new Map<string, string[]>(nodeIds.map((id) => [id, []]))

    for (const edge of edges) {
        adjacency.get(edge.source)?.push(edge.target)
        adjacency.get(edge.target)?.push(edge.source)
    }

    return adjacency
}